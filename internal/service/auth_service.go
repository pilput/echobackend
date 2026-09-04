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
		githubConfig:           config.GitHub,
		frontendConfig:         config.Frontend,
		emailService:           emailService,
		httpClient:             &http.Client{Timeout: 10 * time.Second},
		oauthExchangeCache:     oauthExchangeCache,
		oauthExchangeCodes:     make(map[string]oauthExchangeEntry),
	}
}

func (s *authService) Register(ctx context.Context, email, username, password string) (*model.User, error) {
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

	tokenString, refreshToken, err := s.createTokenAndSession(ctx, user)
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
		if err := s.emailService.EnqueuePasswordResetEmail(email, resetLink); err != nil {
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

	if err := s.sessionRepo.DeleteByUserID(ctx, user.ID); err != nil {
		authLog.Warn("failed to invalidate sessions after password reset", "user_id", user.ID, "error", err)
	}

	s.activityService.LogActivity(ctx, &user.ID, model.ActivityPasswordReset, model.StatusSuccess, ipAddress, userAgent, nil, nil)

	return nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken, ipAddress, userAgent string) (string, string, *model.User, error) {
	refreshTokenHash := tokenHash(refreshToken)
	session, err := s.sessionRepo.GetSessionByRefreshToken(ctx, refreshTokenHash)
	if err != nil {
		return "", "", nil, apperrors.ErrInvalidToken
	}

	if session.ExpiresAt != nil && time.Now().After(*session.ExpiresAt) {
		if err := s.sessionRepo.DeleteSession(ctx, refreshTokenHash); err != nil {
			authLog.Warn("failed to delete expired session", "user_id", session.UserID, "error", err)
		}
		return "", "", nil, apperrors.ErrTokenExpired
	}

	user, err := s.userRepo.GetByID(ctx, session.UserID, false)
	if err != nil {
		return "", "", nil, err
	}

	tokenString, err := s.createAccessToken(user)
	if err != nil {
		return "", "", nil, err
	}

	s.activityService.LogActivity(ctx, &user.ID, model.ActivityTokenRefresh, model.StatusSuccess, ipAddress, userAgent, nil, nil)

	// The refresh token itself is not rotated: the same session/refresh token
	// keeps being reused until it expires, at which point the user must log in
	// again to obtain a new one.
	return tokenString, refreshToken, user, nil
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

	s.activityService.LogActivity(ctx, &userID, model.ActivityPasswordChange, model.StatusSuccess, ipAddress, userAgent, nil, nil)

	return nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	return s.sessionRepo.DeleteSession(ctx, tokenHash(refreshToken))
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

// DeleteAccount soft-deletes the authenticated user's own account.
func (s *authService) DeleteAccount(ctx context.Context, userID string) error {
	return s.userRepo.SoftDeleteByID(ctx, userID)
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
			email = *githubUser.Email
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

	tokenString, refreshToken, err := s.createTokenAndSession(ctx, user)
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
		"user_id":        user.ID,
		"username":       user.Username,
		"email":          user.Email,
		"is_super_admin": user.IsSuperAdmin,
		"iat":            time.Now().Unix(),
		"exp":            time.Now().Add(s.jwtExpiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(s.jwtSecret)
}

func (s *authService) createTokenAndSession(ctx context.Context, user *model.User) (string, string, error) {
	tokenString, err := s.createAccessToken(user)
	if err != nil {
		return "", "", err
	}

	refreshBytes := make([]byte, 64)
	if _, err := rand.Read(refreshBytes); err != nil {
		return "", "", err
	}
	refreshToken := "pl_" + base64.RawURLEncoding.EncodeToString(refreshBytes)

	sess := &model.Session{
		RefreshToken: tokenHash(refreshToken),
		UserID:       user.ID,
		ExpiresAt:    new(time.Now().Add(s.refreshTokenExpiry)),
	}
	if err := s.sessionRepo.CreateSession(ctx, sess); err != nil {
		return "", "", err
	}

	return tokenString, refreshToken, nil
}

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, err
	}
	return b, nil
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
