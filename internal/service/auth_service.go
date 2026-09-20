package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"echobackend/config"
	apperrors "echobackend/internal/apperror"
	"echobackend/internal/model"
	"echobackend/internal/repository"

	pkgpassword "echobackend/pkg/password"
	"echobackend/pkg/uid"
	"echobackend/pkg/validator"

	"github.com/golang-jwt/jwt/v5"
)

type AuthService interface {
	Register(ctx context.Context, email, username, password string) (*model.User, error)
	Login(ctx context.Context, identifier, password, ipAddress, userAgent string) (string, string, *model.User, error)
	ForgotPassword(ctx context.Context, email, ipAddress, userAgent string) error
	ResetPassword(ctx context.Context, token, password, ipAddress, userAgent string) error
	RefreshToken(ctx context.Context, refreshToken, ipAddress, userAgent string) (string, string, *model.User, error)
	ChangePassword(ctx context.Context, userID, currentPassword, newPassword, ipAddress, userAgent string) error
	Logout(ctx context.Context, refreshToken string) error
	RevokeUserSessions(ctx context.Context, targetUserID, actorID, ipAddress, userAgent string) (int64, error)
	RevokeAllSessions(ctx context.Context, actorID, ipAddress, userAgent string) (int64, error)
	GetProfile(ctx context.Context, userID string) (*model.User, error)
	UpdateProfile(ctx context.Context, userID, username, firstName, lastName string) (*model.User, error)
	DeleteAccount(ctx context.Context, userID string) error
	CheckUsernameExists(ctx context.Context, username string) (bool, error)
	GetGithubOAuthURL(state string) string
	GetGithubToken(ctx context.Context, code string) (string, error)
	SignInWithGithub(ctx context.Context, githubUser *GithubUser, ipAddress, userAgent string) (string, string, *model.User, error)
	CreateOAuthExchangeCode(ctx context.Context, accessToken, refreshToken string, user *model.User) (string, error)
	ExchangeOAuthCode(ctx context.Context, code string) (string, string, *model.User, error)
}

type GithubUser struct {
	Login     string  `json:"login"`
	ID        int64   `json:"id"`
	AvatarURL string  `json:"avatar_url"`
	Email     *string `json:"email"`
	Name      string  `json:"name"`
	HTMLURL   string  `json:"html_url"`
}

type EmailSender interface {
	EnqueuePasswordResetEmail(to, resetLink string) error
	IsConfigured() bool
}

type AuthCache interface {
	BuildKey(parts ...string) string
	GetJSONAndDelete(ctx context.Context, key string, dest any) (bool, error)
	SetJSONWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error
}

type authService struct {
	authRepo               repository.AuthRepository
	userRepo               repository.UserRepository
	sessionRepo            repository.SessionRepository
	passwordResetTokenRepo repository.PasswordResetTokenRepository
	activityService        AuthActivityService
	jwtSecret              []byte
	jwtExpiry              time.Duration
	refreshTokenExpiry     time.Duration
	refreshTokenAbsolute   time.Duration
	refreshTokenGrace      time.Duration
	githubConfig           config.GitHubConfig
	frontendConfig         config.FrontendConfig
	emailService           EmailSender
	httpClient             *http.Client
	oauthExchangeCache     AuthCache
	oauthExchangeCodes     map[string]oauthExchangeEntry
	oauthExchangeMu        sync.Mutex
}

const oauthExchangeTTL = 2 * time.Minute

type oauthExchangeEntry struct {
	AccessToken  string
	RefreshToken string
	User         *model.User
	ExpiresAt    time.Time
}

func NewAuthService(
	authRepo repository.AuthRepository,
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	passwordResetTokenRepo repository.PasswordResetTokenRepository,
	activityService AuthActivityService,
	config *config.Config,
	oauthExchangeCache AuthCache,
	emailService EmailSender,
) AuthService {
	return &authService{
		authRepo:               authRepo,
		userRepo:               userRepo,
		sessionRepo:            sessionRepo,
		passwordResetTokenRepo: passwordResetTokenRepo,
		activityService:        activityService,
		jwtSecret:              []byte(config.Auth.JWTSecret),
		jwtExpiry:              config.Auth.JWTExpiry,
		refreshTokenExpiry:     config.Auth.RefreshTokenExpiry,
		refreshTokenAbsolute:   config.Auth.RefreshTokenAbsoluteExpiry,
		refreshTokenGrace:      config.Auth.RefreshTokenGracePeriod,
		githubConfig:           config.GitHub,
		frontendConfig:         config.Frontend,
		emailService:           emailService,
		httpClient:             &http.Client{Timeout: 10 * time.Second},
		oauthExchangeCache:     oauthExchangeCache,
		oauthExchangeCodes:     make(map[string]oauthExchangeEntry),
	}
}

func (s *authService) Register(ctx context.Context, email, username, password string) (*model.User, error) {
	email = normalizeEmail(email)
	_, err := s.authRepo.FindUserByEmail(ctx, email)
	if err == nil {
		return nil, apperrors.ErrUserExists
	}
	if err != nil && !errors.Is(err, apperrors.ErrUserNotFound) {
		return nil, err
	}

	err = s.userRepo.CheckUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	hashedPassword, err := pkgpassword.Hash(password)
	if err != nil {
		return nil, err
	}

	newUser := &model.User{
		Email:    email,
		Username: &username,
		Password: &hashedPassword,
	}

	if err := s.authRepo.CreateUser(ctx, newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

func (s *authService) Login(ctx context.Context, identifier, password, ipAddress, userAgent string) (string, string, *model.User, error) {
	user, err := s.authRepo.FindUserByIdentifier(ctx, identifier)
	if err != nil {
		s.activityService.LogActivity(ctx, nil, model.ActivityLoginFailed, model.StatusFailure, ipAddress, userAgent, nil, nil)
		return "", "", nil, apperrors.ErrInvalidCredentials
	}

	if user.Password == nil {
		s.activityService.LogActivity(ctx, &user.ID, model.ActivityLoginFailed, model.StatusFailure, ipAddress, userAgent, nil, nil)
		return "", "", nil, apperrors.ErrInvalidCredentials
	}

	matched, compareErr := pkgpassword.Compare(*user.Password, password)
	if compareErr != nil || !matched {
		s.activityService.LogActivity(ctx, &user.ID, model.ActivityLoginFailed, model.StatusFailure, ipAddress, userAgent, nil, nil)
		return "", "", nil, apperrors.ErrInvalidCredentials
	}

	// Transparent upgrade: rehash legacy bcrypt or outdated argon2 hashes to current Argon2id defaults
	if pkgpassword.NeedsRehash(*user.Password) {
		if newHash, err := pkgpassword.Hash(password); err == nil {
			user.Password = &newHash
		}
	}

	tokenString, refreshToken, err := s.createTokenAndSession(ctx, user, ipAddress, userAgent)
	if err != nil {
		return "", "", nil, err
	}

	s.activityService.LogActivity(ctx, &user.ID, model.ActivityLogin, model.StatusSuccess, ipAddress, userAgent, nil, nil)

	now := time.Now()
	user.LastLoggedAt = &now
	if err := s.userRepo.Update(ctx, user); err != nil {
		authLog.Warn("failed to update last_logged_at", "user_id", user.ID, "error", err)
	}

	return tokenString, refreshToken, user, nil
}

func (s *authService) ForgotPassword(ctx context.Context, email, ipAddress, userAgent string) error {
	user, err := s.authRepo.FindUserByEmail(ctx, email)
	if err != nil {
		// Intentionally swallow the error (including not-found) so the response
		// never reveals whether an account exists for the given email.
		return nil //nolint:nilerr // user-enumeration protection
	}

	resetBytes, err := generateRandomBytes(32)
	if err != nil {
		return err
	}
	resetToken := "pr_" + base64.RawURLEncoding.EncodeToString(resetBytes)
	expiresAt := time.Now().Add(1 * time.Hour)

	tokenEntry := &model.PasswordResetToken{
		UserID:    user.ID,
		Token:     tokenHash(resetToken),
		ExpiresAt: expiresAt,
	}

	if err := s.passwordResetTokenRepo.DeleteByUserID(ctx, user.ID); err != nil {
		// Best-effort cleanup of stale reset tokens. A failure here does not
		// block issuing a new one, but is logged so drift is observable.
		authLog.Warn("failed to delete previous password reset tokens", "user_id", user.ID, "error", err)
	}

	if err := s.passwordResetTokenRepo.Create(ctx, tokenEntry); err != nil {
		return err
	}

	resetLink := buildPasswordResetLink(s.frontendConfig.ResetPasswordURL, resetToken)
	if s.emailService != nil && s.emailService.IsConfigured() {
		if err := s.emailService.EnqueuePasswordResetEmail(user.Email, resetLink); err != nil {
			errMsg := "Failed to queue email"
			s.activityService.LogActivity(ctx, &user.ID, model.ActivityPasswordResetReq, model.StatusFailure, ipAddress, userAgent, &errMsg, nil)
			authLog.Error("failed to queue password reset email", "error", err, "user_id", user.ID)
			return nil
		}
		s.activityService.LogActivity(ctx, &user.ID, model.ActivityPasswordResetReq, model.StatusSuccess, ipAddress, userAgent, nil, map[string]any{"emailQueued": true})
		return nil
	}

	s.activityService.LogActivity(ctx, &user.ID, model.ActivityPasswordResetReq, model.StatusSuccess, ipAddress, userAgent, nil, map[string]any{"devMode": true})

	return nil
}

func (s *authService) ResetPassword(ctx context.Context, token, password, ipAddress, userAgent string) error {
	tokenEntry, err := s.passwordResetTokenRepo.FindByToken(ctx, tokenHash(token))
	if err != nil {
		return apperrors.ErrInvalidToken
	}
	if tokenEntry == nil {
		return apperrors.ErrInvalidToken
	}

	if tokenEntry.UsedAt != nil {
		return apperrors.ErrPasswordResetTokenUsed
	}

	if time.Now().After(tokenEntry.ExpiresAt) {
		return apperrors.ErrPasswordResetTokenExpired
	}

	user, err := s.userRepo.GetByID(ctx, tokenEntry.UserID, false)
	if err != nil {
		return apperrors.ErrUserNotFound
	}

	hashedPassword, err := pkgpassword.Hash(password)
	if err != nil {
		return err
	}

	user.Password = &hashedPassword
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	if err := s.passwordResetTokenRepo.MarkUsed(ctx, tokenEntry.ID); err != nil {
		// The password was already changed; a failure to mark the token used is
		// logged so a potentially reusable token is observable. Sessions are
		// still invalidated below.
		authLog.Warn("failed to mark password reset token as used", "token_id", tokenEntry.ID, "error", err)
	}

	if _, err := s.sessionRepo.DeleteByUserID(ctx, user.ID); err != nil {
		authLog.Warn("failed to invalidate sessions after password reset", "user_id", user.ID, "error", err)
	}

	s.activityService.LogActivity(ctx, &user.ID, model.ActivityPasswordReset, model.StatusSuccess, ipAddress, userAgent, nil, nil)

	return nil
}

// RefreshToken exchanges a refresh token for a fresh access token and a fresh
// refresh token, rotating the chain forward (RFC 9700 §4.14).
//
// The returned refresh token always replaces the one that was sent: callers
// must persist it. Rotation is what makes a stolen token detectable — a token
// presented after it was already exchanged is treated as a replay and takes the
// entire family down with it.
func (s *authService) RefreshToken(ctx context.Context, refreshToken, ipAddress, userAgent string) (string, string, *model.User, error) {
	refreshTokenHash := tokenHash(refreshToken)
	session, err := s.sessionRepo.GetSessionByRefreshToken(ctx, refreshTokenHash)
	if err != nil || session == nil {
		return "", "", nil, apperrors.ErrInvalidToken
	}

	now := time.Now()

	// Checked before anything else: once a family is past its maximum lifetime
	// nothing in it can be revived, not even through the grace window below.
	if now.After(session.AbsoluteExpiresAt) {
		// ASVS 7.3.2 — drop the whole family at once, since every row in it
		// shares this deadline.
		if err := s.sessionRepo.DeleteByFamilyID(ctx, session.FamilyID); err != nil {
			authLog.Warn("failed to delete session family past absolute expiry", "user_id", session.UserID, "error", err)
		}
		return "", "", nil, apperrors.ErrTokenExpired
	}

	if session.Rotated() {
		// Inside the grace window this is almost certainly the client's own
		// concurrent refresh, not an attacker, so it is served rather than
		// punished. Grace is anchored to the first rotation, so replaying the
		// same token cannot keep pushing the window forward.
		if s.refreshTokenGrace > 0 && !now.After(session.RotatedAt.Add(s.refreshTokenGrace)) {
			return s.issueRotatedToken(ctx, session, nil, ipAddress, userAgent)
		}

		// Past the window, assume the token leaked: whoever holds the successor
		// may be the attacker, so the whole chain goes.
		if err := s.sessionRepo.DeleteByFamilyID(ctx, session.FamilyID); err != nil {
			authLog.Error("failed to revoke session family after refresh token reuse",
				"user_id", session.UserID, "family_id", session.FamilyID, "error", err)
		}
		s.activityService.LogActivity(ctx, &session.UserID, model.ActivityTokenReuse, model.StatusFailure,
			ipAddress, userAgent, nil, map[string]any{"family_id": session.FamilyID})
		authLog.Warn("refresh token reuse detected, session family revoked",
			"user_id", session.UserID, "family_id", session.FamilyID)
		return "", "", nil, apperrors.ErrInvalidToken
	}

	if now.After(session.ExpiresAt) {
		// Only this leaf is dead. Sibling tokens minted during a grace window
		// carry their own deadlines, so the family is left alone.
		if err := s.sessionRepo.DeleteSession(ctx, refreshTokenHash); err != nil {
			authLog.Warn("failed to delete expired session", "user_id", session.UserID, "error", err)
		}
		return "", "", nil, apperrors.ErrTokenExpired
	}

	return s.issueRotatedToken(ctx, session, session, ipAddress, userAgent)
}

// issueRotatedToken mints the successor of current and returns it along with a
// fresh access token.
//
// When rotateFrom is non-nil the successor replaces it atomically; when it is
// nil (the grace path) the successor is inserted as a sibling and the parent is
// left as it is, since it was already rotated by the request that won the race.
func (s *authService) issueRotatedToken(
	ctx context.Context,
	current *model.Session,
	rotateFrom *model.Session,
	ipAddress, userAgent string,
) (string, string, *model.User, error) {
	user, err := s.userRepo.GetByID(ctx, current.UserID, false)
	if errors.Is(err, apperrors.ErrUserNotFound) {
		// The account was deleted after the session was issued.
		return "", "", nil, apperrors.ErrInvalidToken
	}
	if err != nil {
		return "", "", nil, err
	}

	accessToken, err := s.createAccessToken(user)
	if err != nil {
		return "", "", nil, err
	}

	nextToken, next, err := s.buildSession(user.ID, current.FamilyID, current.AbsoluteExpiresAt, ipAddress, userAgent)
	if err != nil {
		return "", "", nil, err
	}

	if rotateFrom != nil {
		err = s.sessionRepo.RotateSession(ctx, rotateFrom.ID, next)
		if errors.Is(err, apperrors.ErrSessionAlreadyRotated) {
			// A concurrent refresh rotated this row between our read and our
			// write. With a grace window configured that is the same benign
			// race handled above, so mint a sibling instead of failing.
			if s.refreshTokenGrace <= 0 {
				return "", "", nil, apperrors.ErrInvalidToken
			}
			err = s.sessionRepo.CreateSession(ctx, next)
		}
	} else {
		err = s.sessionRepo.CreateSession(ctx, next)
	}
	if err != nil {
		return "", "", nil, err
	}

	s.pruneExpiredSessions(ctx, user.ID)
	s.activityService.LogActivity(ctx, &user.ID, model.ActivityTokenRefresh, model.StatusSuccess, ipAddress, userAgent, nil, nil)

	return accessToken, nextToken, user, nil
}

// pruneExpiredSessions drops the user's dead rows. Rotation appends a row per
// refresh, so without this the table would grow with every access-token
// renewal. Best-effort: a failure here must not fail the refresh.
func (s *authService) pruneExpiredSessions(ctx context.Context, userID string) {
	if _, err := s.sessionRepo.DeleteExpired(ctx, userID); err != nil {
		authLog.Warn("failed to prune expired sessions", "user_id", userID, "error", err)
	}
}

func (s *authService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword, ipAddress, userAgent string) error {
	user, err := s.userRepo.GetByID(ctx, userID, false)
	if err != nil {
		return apperrors.ErrUserNotFound
	}

	if user.Password == nil {
		return apperrors.ErrInvalidCredentials
	}

	matched, compareErr := pkgpassword.Compare(*user.Password, currentPassword)
	if compareErr != nil || !matched {
		s.activityService.LogActivity(ctx, &userID, model.ActivityPasswordChange, model.StatusFailure, ipAddress, userAgent, nil, nil)
		return apperrors.ErrInvalidCredentials
	}

	hashedPassword, err := pkgpassword.Hash(newPassword)
	if err != nil {
		return err
	}

	user.Password = &hashedPassword
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// Revoke every refresh token so sessions opened with the old password
	// (possibly by someone else) cannot be extended.
	if _, err := s.sessionRepo.DeleteByUserID(ctx, user.ID); err != nil {
		authLog.Warn("failed to invalidate sessions after password change", "user_id", user.ID, "error", err)
	}

	s.activityService.LogActivity(ctx, &userID, model.ActivityPasswordChange, model.StatusSuccess, ipAddress, userAgent, nil, nil)

	return nil
}

// Logout ends the whole rotation chain the token belongs to, not just the token
// itself, so that any sibling minted during a grace window dies with it
// (ASVS 7.4.1). Unknown tokens are a no-op: logging out is idempotent.
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	session, err := s.sessionRepo.GetSessionByRefreshToken(ctx, tokenHash(refreshToken))
	if err != nil || session == nil {
		return nil //nolint:nilerr // logging out an unknown token is not an error
	}
	return s.sessionRepo.DeleteByFamilyID(ctx, session.FamilyID)
}

// RevokeUserSessions terminates every session belonging to one user and
// reports how many were killed (OWASP ASVS v5.0 7.4.5).
//
// The caller is expected to be an administrator; authorisation is enforced by
// the route's middleware, not here. actorID is recorded so the activity log
// shows who did it, not just that it happened.
//
// Access tokens already issued are not affected and stay valid until they
// expire, so the user keeps API access for up to the JWT lifetime.
func (s *authService) RevokeUserSessions(ctx context.Context, targetUserID, actorID, ipAddress, userAgent string) (int64, error) {
	if !validator.IsValidUUID(targetUserID) {
		return 0, apperrors.ErrInvalidUserID
	}

	// Look the user up first so a typo in the id is a 404 rather than a
	// success that silently revoked nothing.
	if _, err := s.userRepo.GetByID(ctx, targetUserID, false); err != nil {
		return 0, err
	}

	revoked, err := s.sessionRepo.DeleteByUserID(ctx, targetUserID)
	if err != nil {
		return 0, err
	}

	s.activityService.LogActivity(ctx, &targetUserID, model.ActivitySessionRevoked, model.StatusSuccess,
		ipAddress, userAgent, nil, map[string]any{"revoked_by": actorID, "revoked_sessions": revoked})
	authLog.Info("administrator revoked user sessions",
		"target_user_id", targetUserID, "actor_id", actorID, "revoked_sessions", revoked)

	return revoked, nil
}

// RevokeAllSessions terminates every session of every user — the second half of
// ASVS 7.4.5, for when a leak is suspected but its blast radius is not known.
//
// This logs out the acting administrator too. It is deliberately not something
// any other code path calls.
func (s *authService) RevokeAllSessions(ctx context.Context, actorID, ipAddress, userAgent string) (int64, error) {
	revoked, err := s.sessionRepo.DeleteAll(ctx)
	if err != nil {
		return 0, err
	}

	// user_id is left nil: this event belongs to no single account.
	s.activityService.LogActivity(ctx, nil, model.ActivitySessionRevoked, model.StatusSuccess,
		ipAddress, userAgent, nil, map[string]any{"revoked_by": actorID, "revoked_sessions": revoked, "scope": "all_users"})
	authLog.Warn("administrator revoked every session", "actor_id", actorID, "revoked_sessions", revoked)

	return revoked, nil
}

func (s *authService) GetProfile(ctx context.Context, userID string) (*model.User, error) {
	return s.userRepo.GetByID(ctx, userID, false)
}

// UpdateProfile updates the authenticated user's username/first name/last name.
// When the username changes, it is checked for uniqueness against other users.
func (s *authService) UpdateProfile(ctx context.Context, userID, username, firstName, lastName string) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID, false)
	if err != nil {
		return nil, err
	}

	if user.Username == nil || *user.Username != username {
		if err := s.userRepo.CheckUserByUsername(ctx, username); err != nil {
			return nil, err
		}
	}

	user.Username = &username
	user.FirstName = &firstName
	user.LastName = &lastName

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// DeleteAccount soft-deletes the authenticated user's own account and revokes
// all of its refresh tokens.
func (s *authService) DeleteAccount(ctx context.Context, userID string) error {
	if err := s.userRepo.SoftDeleteByID(ctx, userID); err != nil {
		return err
	}
	if _, err := s.sessionRepo.DeleteByUserID(ctx, userID); err != nil {
		authLog.Warn("failed to invalidate sessions after account deletion", "user_id", userID, "error", err)
	}
	return nil
}

// CheckUsernameExists reports whether a user with the given username already exists.
func (s *authService) CheckUsernameExists(ctx context.Context, username string) (bool, error) {
	err := s.userRepo.CheckUserByUsername(ctx, username)
	if err == nil {
		return false, nil
	}
	if errors.Is(err, apperrors.ErrUserExists) {
		return true, nil
	}
	return false, err
}

func (s *authService) GetGithubOAuthURL(state string) string {
	authURL, _ := url.Parse("https://github.com/login/oauth/authorize")
	q := authURL.Query()
	q.Set("client_id", s.githubConfig.ClientID)
	q.Set("redirect_uri", s.githubConfig.RedirectURI)
	q.Set("scope", "user:email")
	q.Set("state", state)
	authURL.RawQuery = q.Encode()
	return authURL.String()
}

func (s *authService) GetGithubToken(ctx context.Context, code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", s.githubConfig.ClientID)
	data.Set("client_secret", s.githubConfig.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", s.githubConfig.RedirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	accessToken := values.Get("access_token")
	if accessToken == "" {
		return "", errors.New("no access_token in GitHub response")
	}

	return accessToken, nil
}

func (s *authService) SignInWithGithub(ctx context.Context, githubUser *GithubUser, ipAddress, userAgent string) (string, string, *model.User, error) {
	var user *model.User

	githubID := githubUser.ID
	user, err := s.authRepo.FindUserByGithubID(ctx, githubID)
	if err != nil && !errors.Is(err, apperrors.ErrUserNotFound) {
		s.activityService.LogActivity(ctx, nil, model.ActivityOAuthLoginFailed, model.StatusFailure, ipAddress, userAgent, nil, map[string]any{"provider": "github"})
		return "", "", nil, err
	}

	if user == nil || errors.Is(err, apperrors.ErrUserNotFound) {
		email := ""
		if githubUser.Email != nil {
			email = normalizeEmail(*githubUser.Email)
		} else {
			email = fmt.Sprintf("%d@github.placeholder", githubUser.ID)
		}

		username := githubUser.Login
		newUser := &model.User{
			Email:    email,
			Username: &username,
			GithubID: &githubID,
			Image:    &githubUser.AvatarURL,
		}

		if err := s.authRepo.CreateUser(ctx, newUser); err != nil {
			s.activityService.LogActivity(ctx, nil, model.ActivityOAuthLoginFailed, model.StatusFailure, ipAddress, userAgent, nil, map[string]any{"provider": "github", "error": err.Error()})
			return "", "", nil, err
		}
		user = newUser
	}

	tokenString, refreshToken, err := s.createTokenAndSession(ctx, user, ipAddress, userAgent)
	if err != nil {
		s.activityService.LogActivity(ctx, &user.ID, model.ActivityOAuthLoginFailed, model.StatusFailure, ipAddress, userAgent, nil, map[string]any{"provider": "github"})
		return "", "", nil, err
	}

	s.activityService.LogActivity(ctx, &user.ID, model.ActivityOAuthLogin, model.StatusSuccess, ipAddress, userAgent, nil, map[string]any{"provider": "github"})

	now := time.Now()
	user.LastLoggedAt = &now
	if err := s.userRepo.Update(ctx, user); err != nil {
		authLog.Warn("failed to update last_logged_at", "user_id", user.ID, "error", err)
	}

	return tokenString, refreshToken, user, nil
}

func (s *authService) CreateOAuthExchangeCode(ctx context.Context, accessToken, refreshToken string, user *model.User) (string, error) {
	if user == nil {
		return "", errors.New("oauth exchange user is nil")
	}

	codeBytes, err := generateRandomBytes(32)
	if err != nil {
		return "", err
	}
	code := "oc_" + base64.RawURLEncoding.EncodeToString(codeBytes)
	entry := oauthExchangeEntry{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
		ExpiresAt:    time.Now().Add(oauthExchangeTTL),
	}

	if s.oauthExchangeCache != nil {
		key := s.oauthExchangeCache.BuildKey("oauth_exchange", code)
		if err := s.oauthExchangeCache.SetJSONWithTTL(ctx, key, entry, oauthExchangeTTL); err != nil {
			return "", err
		}
		return code, nil
	}

	s.oauthExchangeMu.Lock()
	defer s.oauthExchangeMu.Unlock()
	s.cleanupExpiredOAuthExchangeCodesLocked(time.Now())
	s.oauthExchangeCodes[code] = entry

	return code, nil
}

func (s *authService) ExchangeOAuthCode(ctx context.Context, code string) (string, string, *model.User, error) {
	if s.oauthExchangeCache != nil {
		key := s.oauthExchangeCache.BuildKey("oauth_exchange", code)
		var entry oauthExchangeEntry
		found, err := s.oauthExchangeCache.GetJSONAndDelete(ctx, key, &entry)
		if err != nil {
			return "", "", nil, err
		}
		if !found || time.Now().After(entry.ExpiresAt) || entry.User == nil {
			return "", "", nil, apperrors.ErrInvalidToken
		}
		return entry.AccessToken, entry.RefreshToken, entry.User, nil
	}

	s.oauthExchangeMu.Lock()
	defer s.oauthExchangeMu.Unlock()

	now := time.Now()
	s.cleanupExpiredOAuthExchangeCodesLocked(now)

	entry, ok := s.oauthExchangeCodes[code]
	if !ok || now.After(entry.ExpiresAt) {
		delete(s.oauthExchangeCodes, code)
		return "", "", nil, apperrors.ErrInvalidToken
	}

	delete(s.oauthExchangeCodes, code)
	return entry.AccessToken, entry.RefreshToken, entry.User, nil
}

func (s *authService) cleanupExpiredOAuthExchangeCodesLocked(now time.Time) {
	for code, entry := range s.oauthExchangeCodes {
		if now.After(entry.ExpiresAt) {
			delete(s.oauthExchangeCodes, code)
		}
	}
}

func (s *authService) createAccessToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(s.jwtExpiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(s.jwtSecret)
}

// createTokenAndSession starts a new rotation chain: the session it creates is
// the root of its own family and sets the absolute deadline every later
// rotation inherits.
func (s *authService) createTokenAndSession(ctx context.Context, user *model.User, ipAddress, userAgent string) (string, string, error) {
	tokenString, err := s.createAccessToken(user)
	if err != nil {
		return "", "", err
	}

	absoluteExpiresAt := time.Now().Add(s.refreshTokenAbsolute)
	refreshToken, sess, err := s.buildSession(user.ID, "", absoluteExpiresAt, ipAddress, userAgent)
	if err != nil {
		return "", "", err
	}

	if err := s.sessionRepo.CreateSession(ctx, sess); err != nil {
		return "", "", err
	}

	s.pruneExpiredSessions(ctx, user.ID)

	return tokenString, refreshToken, nil
}

// buildSession mints a refresh token and the row that will hold its hash,
// returning the raw token (the only time it exists in plaintext) and the
// unsaved session.
//
// An empty familyID starts a new family, in which case the row's own id is used
// so the root belongs to the family it heads.
func (s *authService) buildSession(
	userID, familyID string,
	absoluteExpiresAt time.Time,
	ipAddress, userAgent string,
) (string, *model.Session, error) {
	refreshBytes, err := generateRandomBytes(64)
	if err != nil {
		return "", nil, err
	}
	refreshToken := "pl_" + base64.RawURLEncoding.EncodeToString(refreshBytes)

	now := time.Now()
	// The sliding window never outlives the absolute cap, so a chain that is
	// refreshed right before the deadline does not gain extra time from it.
	expiresAt := now.Add(s.refreshTokenExpiry)
	if expiresAt.After(absoluteExpiresAt) {
		expiresAt = absoluteExpiresAt
	}

	id, err := uid.NewV7()
	if err != nil {
		return "", nil, err
	}
	if familyID == "" {
		familyID = id
	}

	sess := &model.Session{
		ID:                id,
		FamilyID:          familyID,
		RefreshToken:      tokenHash(refreshToken),
		UserID:            userID,
		CreatedAt:         now,
		ExpiresAt:         expiresAt,
		AbsoluteExpiresAt: absoluteExpiresAt,
	}
	if ipAddress != "" {
		sess.IPAddress = &ipAddress
	}
	if userAgent != "" {
		sess.UserAgent = &userAgent
	}

	return refreshToken, sess, nil
}

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, err
	}
	return b, nil
}

// normalizeEmail canonicalizes an email address before it is stored, so
// addresses that differ only in case or surrounding spaces map to one account.
// Lookups additionally compare with LOWER(email) to match legacy rows.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func buildPasswordResetLink(baseURL, token string) string {
	if baseURL == "" {
		baseURL = "http://localhost:3000/reset-password"
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return baseURL + "?token=" + url.QueryEscape(token)
	}

	q := parsed.Query()
	q.Set("token", token)
	parsed.RawQuery = q.Encode()
	return parsed.String()
}
