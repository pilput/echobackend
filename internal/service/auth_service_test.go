package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"echobackend/config"
	apperrors "echobackend/internal/apperror"
	"echobackend/internal/model"
	pkgpassword "echobackend/pkg/password"

	"golang.org/x/crypto/bcrypt"
)

type mockAuthRepo struct {
	findUserByEmailFn      func(ctx context.Context, email string) (*model.User, error)
	findUserByIdentifierFn func(ctx context.Context, identifier string) (*model.User, error)
	findUserByGithubIDFn   func(ctx context.Context, githubID int64) (*model.User, error)
	createUserFn           func(ctx context.Context, user *model.User) error
}

func (m *mockAuthRepo) FindUserByEmail(ctx context.Context, email string) (*model.User, error) {
	if m.findUserByEmailFn != nil {
		return m.findUserByEmailFn(ctx, email)
	}
	return nil, apperrors.ErrUserNotFound
}

func (m *mockAuthRepo) FindUserByIdentifier(ctx context.Context, identifier string) (*model.User, error) {
	if m.findUserByIdentifierFn != nil {
		return m.findUserByIdentifierFn(ctx, identifier)
	}
	return nil, apperrors.ErrUserNotFound
}

func (m *mockAuthRepo) FindUserByGithubID(ctx context.Context, githubID int64) (*model.User, error) {
	if m.findUserByGithubIDFn != nil {
		return m.findUserByGithubIDFn(ctx, githubID)
	}
	return nil, apperrors.ErrUserNotFound
}

func (m *mockAuthRepo) CreateUser(ctx context.Context, user *model.User) error {
	if m.createUserFn != nil {
		return m.createUserFn(ctx, user)
	}
	return nil
}

type mockSessionRepo struct {
	createSessionFn            func(ctx context.Context, session *model.Session) error
	getSessionByRefreshTokenFn func(ctx context.Context, tokenHash string) (*model.Session, error)
	rotateSessionFn            func(ctx context.Context, currentID string, next *model.Session) error
	deleteSessionFn            func(ctx context.Context, tokenHash string) error
	deleteByUserIDFn           func(ctx context.Context, userID string) (int64, error)
	deleteAllFn                func(ctx context.Context) (int64, error)
	deleteByFamilyIDFn         func(ctx context.Context, familyID string) error
	deleteExpiredFn            func(ctx context.Context, userID string) (int64, error)
}

func (m *mockSessionRepo) CreateSession(ctx context.Context, session *model.Session) error {
	if m.createSessionFn != nil {
		return m.createSessionFn(ctx, session)
	}
	return nil
}

func (m *mockSessionRepo) GetSessionByRefreshToken(ctx context.Context, tokenHash string) (*model.Session, error) {
	if m.getSessionByRefreshTokenFn != nil {
		return m.getSessionByRefreshTokenFn(ctx, tokenHash)
	}
	return nil, nil
}

func (m *mockSessionRepo) DeleteSession(ctx context.Context, tokenHash string) error {
	if m.deleteSessionFn != nil {
		return m.deleteSessionFn(ctx, tokenHash)
	}
	return nil
}

func (m *mockSessionRepo) DeleteByUserID(ctx context.Context, userID string) (int64, error) {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(ctx, userID)
	}
	return 0, nil
}

func (m *mockSessionRepo) DeleteAll(ctx context.Context) (int64, error) {
	if m.deleteAllFn != nil {
		return m.deleteAllFn(ctx)
	}
	return 0, nil
}

func (m *mockSessionRepo) UpdateSession(ctx context.Context, s *model.Session) error {
	return nil
}

func (m *mockSessionRepo) RotateSession(ctx context.Context, currentID string, next *model.Session) error {
	if m.rotateSessionFn != nil {
		return m.rotateSessionFn(ctx, currentID, next)
	}
	return nil
}

func (m *mockSessionRepo) DeleteByFamilyID(ctx context.Context, familyID string) error {
	if m.deleteByFamilyIDFn != nil {
		return m.deleteByFamilyIDFn(ctx, familyID)
	}
	return nil
}

func (m *mockSessionRepo) DeleteExpired(ctx context.Context, userID string) (int64, error) {
	if m.deleteExpiredFn != nil {
		return m.deleteExpiredFn(ctx, userID)
	}
	return 0, nil
}

type mockActivityService struct {
	logActivityFn func(ctx context.Context, userID *string, activityType, status, ipAddress, userAgent string, errorMessage *string, metadata map[string]any)
}

func (m *mockActivityService) LogActivity(ctx context.Context, userID *string, activityType, status, ipAddress, userAgent string, errorMessage *string, metadata map[string]any) {
	if m.logActivityFn != nil {
		m.logActivityFn(ctx, userID, activityType, status, ipAddress, userAgent, errorMessage, metadata)
	}
}

func (m *mockActivityService) GetActivityLogs(ctx context.Context, userID, activityType string, limit, offset int) ([]*model.AuthActivityLog, int64, error) {
	return nil, 0, nil
}

func (m *mockActivityService) GetRecentActivity(ctx context.Context, userID string, limit int) ([]*model.AuthActivityLog, error) {
	return nil, nil
}

func (m *mockActivityService) GetFailedLogins(ctx context.Context, since time.Time, limit, offset int) ([]*model.AuthActivityLog, int64, error) {
	return nil, 0, nil
}

func testAuthConfig() *config.Config {
	return &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:                  "01234567890123456789012345678901", // >= 32 chars
			JWTExpiry:                  15 * time.Minute,
			RefreshTokenExpiry:         3 * 24 * time.Hour,
			RefreshTokenAbsoluteExpiry: 30 * 24 * time.Hour,
			RefreshTokenGracePeriod:    60 * time.Second,
		},
	}
}

func TestAuthService_Register_UsesArgon2id(t *testing.T) {
	var createdUser *model.User
	authRepo := &mockAuthRepo{
		createUserFn: func(ctx context.Context, user *model.User) error {
			createdUser = user
			return nil
		},
	}
	userRepo := &mockUserRepo{
		checkUserByUsernameFn: func(ctx context.Context, username string) error {
			return nil
		},
		existsFn: func(ctx context.Context, email string) (bool, error) {
			return false, nil
		},
	}

	svc := NewAuthService(authRepo, userRepo, &mockSessionRepo{}, nil, &mockActivityService{}, testAuthConfig(), nil, nil)

	u, err := svc.Register(context.Background(), "test@example.com", "testuser", "SecurePass123!")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if u == nil || createdUser == nil || createdUser.Password == nil {
		t.Fatalf("expected created user with non-nil password")
	}

	if !strings.HasPrefix(*createdUser.Password, "$argon2id$v=19$") {
		t.Errorf("expected Argon2id hash prefix, got %s", *createdUser.Password)
	}

	match, err := pkgpassword.Compare(*createdUser.Password, "SecurePass123!")
	if err != nil || !match {
		t.Errorf("expected password to match Argon2id hash, match=%v, err=%v", match, err)
	}
}

func TestAuthService_Register_NormalizesEmail(t *testing.T) {
	var lookedUp string
	var createdUser *model.User
	authRepo := &mockAuthRepo{
		findUserByEmailFn: func(ctx context.Context, email string) (*model.User, error) {
			lookedUp = email
			return nil, apperrors.ErrUserNotFound
		},
		createUserFn: func(ctx context.Context, user *model.User) error {
			createdUser = user
			return nil
		},
	}

	svc := NewAuthService(authRepo, &mockUserRepo{}, &mockSessionRepo{}, nil, &mockActivityService{}, testAuthConfig(), nil, nil)
	if _, err := svc.Register(context.Background(), "  Test@Example.COM ", "testuser", "SecurePass123!"); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if lookedUp != "test@example.com" || createdUser == nil || createdUser.Email != "test@example.com" {
		t.Fatalf("expected normalized email, looked up %q, created %+v", lookedUp, createdUser)
	}
}

func TestAuthService_Login_Argon2id_Success(t *testing.T) {
	rawPassword := "ValidPassword123!"
	argonHash, err := pkgpassword.Hash(rawPassword)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := &model.User{
		ID:       "u-argon-1",
		Email:    "argon@example.com",
		Password: &argonHash,
	}

	authRepo := &mockAuthRepo{
		findUserByIdentifierFn: func(ctx context.Context, identifier string) (*model.User, error) {
			return user, nil
		},
	}
	userRepo := &mockUserRepo{
		updateFn: func(ctx context.Context, u *model.User) error {
			return nil
		},
	}
	sessionRepo := &mockSessionRepo{
		createSessionFn: func(ctx context.Context, s *model.Session) error {
			return nil
		},
	}

	svc := NewAuthService(authRepo, userRepo, sessionRepo, nil, &mockActivityService{}, testAuthConfig(), nil, nil)

	token, refresh, loggedUser, err := svc.Login(context.Background(), "argon@example.com", rawPassword, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if token == "" || refresh == "" || loggedUser.ID != "u-argon-1" {
		t.Errorf("unexpected login result: token=%q, refresh=%q", token, refresh)
	}
}

func TestAuthService_Login_Bcrypt_TransparentUpgrade(t *testing.T) {
	rawPassword := "OldBcryptPassword123!"
	bcryptBytes, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}
	bcryptHash := string(bcryptBytes)

	user := &model.User{
		ID:       "u-bcrypt-1",
		Email:    "bcrypt@example.com",
		Password: &bcryptHash,
	}

	var updatedUser *model.User
	authRepo := &mockAuthRepo{
		findUserByIdentifierFn: func(ctx context.Context, identifier string) (*model.User, error) {
			return user, nil
		},
	}
	userRepo := &mockUserRepo{
		updateFn: func(ctx context.Context, u *model.User) error {
			updatedUser = u
			return nil
		},
	}
	sessionRepo := &mockSessionRepo{
		createSessionFn: func(ctx context.Context, s *model.Session) error {
			return nil
		},
	}

	svc := NewAuthService(authRepo, userRepo, sessionRepo, nil, &mockActivityService{}, testAuthConfig(), nil, nil)

	token, refresh, loggedUser, err := svc.Login(context.Background(), "bcrypt@example.com", rawPassword, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if token == "" || refresh == "" || loggedUser.ID != "u-bcrypt-1" {
		t.Errorf("unexpected login result: token=%q, refresh=%q", token, refresh)
	}

	// Verify transparent upgrade happened!
	if updatedUser == nil || updatedUser.Password == nil {
		t.Fatalf("expected userRepo.Update to be called with updated user")
	}

	if !strings.HasPrefix(*updatedUser.Password, "$argon2id$v=19$") {
		t.Errorf("expected upgraded password to be Argon2id, got: %s", *updatedUser.Password)
	}

	// Verify new Argon2id password matches original plaintext password
	match, err := pkgpassword.Compare(*updatedUser.Password, rawPassword)
	if err != nil || !match {
		t.Errorf("upgraded hash verification failed: match=%v, err=%v", match, err)
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	rawPassword := "CorrectPassword123!"
	argonHash, err := pkgpassword.Hash(rawPassword)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := &model.User{
		ID:       "u-wrong-1",
		Email:    "wrong@example.com",
		Password: &argonHash,
	}

	authRepo := &mockAuthRepo{
		findUserByIdentifierFn: func(ctx context.Context, identifier string) (*model.User, error) {
			return user, nil
		},
	}

	svc := NewAuthService(authRepo, &mockUserRepo{}, &mockSessionRepo{}, nil, &mockActivityService{}, testAuthConfig(), nil, nil)

	_, _, _, err = svc.Login(context.Background(), "wrong@example.com", "WrongPassword999!", "127.0.0.1", "test-agent")
	if err == nil || !errors.Is(err, apperrors.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestAuthService_ChangePassword_RevokesSessions(t *testing.T) {
	rawPassword := "CurrentPassword123!"
	argonHash, err := pkgpassword.Hash(rawPassword)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	userRepo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id string, deletedOnly bool) (*model.User, error) {
			return &model.User{ID: id, Password: &argonHash}, nil
		},
	}
	var revokedFor string
	sessionRepo := &mockSessionRepo{
		deleteByUserIDFn: func(ctx context.Context, userID string) (int64, error) {
			revokedFor = userID
			return 1, nil
		},
	}

	svc := NewAuthService(&mockAuthRepo{}, userRepo, sessionRepo, nil, &mockActivityService{}, testAuthConfig(), nil, nil)
	if err := svc.ChangePassword(context.Background(), "u-1", rawPassword, "NewPassword456!", "127.0.0.1", "test-agent"); err != nil {
		t.Fatalf("ChangePassword failed: %v", err)
	}
	if revokedFor != "u-1" {
		t.Fatalf("expected sessions of u-1 to be revoked, got %q", revokedFor)
	}
}

func TestAuthService_DeleteAccount_RevokesSessions(t *testing.T) {
	var revokedFor string
	sessionRepo := &mockSessionRepo{
		deleteByUserIDFn: func(ctx context.Context, userID string) (int64, error) {
			revokedFor = userID
			return 1, nil
		},
	}

	svc := NewAuthService(&mockAuthRepo{}, &mockUserRepo{}, sessionRepo, nil, &mockActivityService{}, testAuthConfig(), nil, nil)
	if err := svc.DeleteAccount(context.Background(), "u-1"); err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}
	if revokedFor != "u-1" {
		t.Fatalf("expected sessions of u-1 to be revoked, got %q", revokedFor)
	}
}

func TestAuthService_RefreshToken_DeletedUserIsInvalidToken(t *testing.T) {
	sessionRepo := &mockSessionRepo{
		getSessionByRefreshTokenFn: func(ctx context.Context, tokenHash string) (*model.Session, error) {
			return liveSession("u-deleted"), nil
		},
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id string, deletedOnly bool) (*model.User, error) {
			return nil, apperrors.ErrUserNotFound
		},
	}

	svc := NewAuthService(&mockAuthRepo{}, userRepo, sessionRepo, nil, &mockActivityService{}, testAuthConfig(), nil, nil)
	_, _, _, err := svc.RefreshToken(context.Background(), "pl_token", "127.0.0.1", "test-agent")
	if !errors.Is(err, apperrors.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got: %v", err)
	}
}

// liveSession builds an unrotated session that is well inside both its sliding
// and its absolute deadline, so tests only have to set what they care about.
func liveSession(userID string) *model.Session {
	now := time.Now()
	return &model.Session{
		ID:                "sess-1",
		FamilyID:          "fam-1",
		RefreshToken:      tokenHash("pl_token"),
		UserID:            userID,
		CreatedAt:         now,
		ExpiresAt:         now.Add(3 * 24 * time.Hour),
		AbsoluteExpiresAt: now.Add(30 * 24 * time.Hour),
	}
}

func refreshTestService(sessionRepo *mockSessionRepo) AuthService {
	userRepo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id string, deletedOnly bool) (*model.User, error) {
			return &model.User{ID: id}, nil
		},
	}
	return NewAuthService(&mockAuthRepo{}, userRepo, sessionRepo, nil, &mockActivityService{}, testAuthConfig(), nil, nil)
}

func TestAuthService_RefreshToken_RotatesAndReturnsNewToken(t *testing.T) {
	var rotatedFrom string
	var next *model.Session
	sessionRepo := &mockSessionRepo{
		getSessionByRefreshTokenFn: func(ctx context.Context, tokenHash string) (*model.Session, error) {
			return liveSession("u-1"), nil
		},
		rotateSessionFn: func(ctx context.Context, currentID string, s *model.Session) error {
			rotatedFrom = currentID
			next = s
			return nil
		},
	}

	svc := refreshTestService(sessionRepo)
	_, newRefresh, _, err := svc.RefreshToken(context.Background(), "pl_token", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}

	if newRefresh == "pl_token" {
		t.Fatal("expected the refresh token to be rotated, got the same value back")
	}
	if rotatedFrom != "sess-1" {
		t.Fatalf("expected session sess-1 to be rotated, got %q", rotatedFrom)
	}
	if next == nil {
		t.Fatal("expected a successor session to be created")
	}
	if next.FamilyID != "fam-1" {
		t.Errorf("expected the successor to stay in family fam-1, got %q", next.FamilyID)
	}
	if next.RefreshToken != tokenHash(newRefresh) {
		t.Error("expected the stored token to be the hash of the returned one")
	}
	if next.RefreshToken == newRefresh {
		t.Error("expected the raw refresh token never to be stored")
	}
}

func TestAuthService_RefreshToken_SlidesExpiryUnderAbsoluteCap(t *testing.T) {
	// A family two hours from its absolute deadline must not gain a full
	// sliding window from being refreshed.
	base := liveSession("u-1")
	base.AbsoluteExpiresAt = time.Now().Add(2 * time.Hour)

	var next *model.Session
	sessionRepo := &mockSessionRepo{
		getSessionByRefreshTokenFn: func(ctx context.Context, tokenHash string) (*model.Session, error) {
			return base, nil
		},
		rotateSessionFn: func(ctx context.Context, currentID string, s *model.Session) error {
			next = s
			return nil
		},
	}

	svc := refreshTestService(sessionRepo)
	if _, _, _, err := svc.RefreshToken(context.Background(), "pl_token", "127.0.0.1", "test-agent"); err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}

	if next == nil {
		t.Fatal("expected a successor session")
	}
	if next.ExpiresAt.After(base.AbsoluteExpiresAt) {
		t.Errorf("sliding expiry %v exceeded the absolute cap %v", next.ExpiresAt, base.AbsoluteExpiresAt)
	}
	if !next.AbsoluteExpiresAt.Equal(base.AbsoluteExpiresAt) {
		t.Error("expected the absolute deadline to be inherited unchanged")
	}
}

func TestAuthService_RefreshToken_ReuseAfterGraceRevokesFamily(t *testing.T) {
	rotatedAt := time.Now().Add(-10 * time.Minute) // well past the 60s grace
	base := liveSession("u-1")
	base.RotatedAt = &rotatedAt

	var revokedFamily string
	var loggedActivity string
	sessionRepo := &mockSessionRepo{
		getSessionByRefreshTokenFn: func(ctx context.Context, tokenHash string) (*model.Session, error) {
			return base, nil
		},
		deleteByFamilyIDFn: func(ctx context.Context, familyID string) error {
			revokedFamily = familyID
			return nil
		},
	}
	activity := &mockActivityService{
		logActivityFn: func(ctx context.Context, userID *string, activityType, status, ip, ua string, errMsg *string, meta map[string]any) {
			loggedActivity = activityType
		},
	}

	svc := NewAuthService(&mockAuthRepo{}, &mockUserRepo{}, sessionRepo, nil, activity, testAuthConfig(), nil, nil)
	_, _, _, err := svc.RefreshToken(context.Background(), "pl_token", "127.0.0.1", "test-agent")
	if !errors.Is(err, apperrors.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken on replay, got: %v", err)
	}
	if revokedFamily != "fam-1" {
		t.Errorf("expected family fam-1 to be revoked, got %q", revokedFamily)
	}
	if loggedActivity != model.ActivityTokenReuse {
		t.Errorf("expected a %q activity log, got %q", model.ActivityTokenReuse, loggedActivity)
	}
}

func TestAuthService_RefreshToken_ReuseWithinGraceIssuesSibling(t *testing.T) {
	rotatedAt := time.Now().Add(-5 * time.Second) // inside the 60s grace
	base := liveSession("u-1")
	base.RotatedAt = &rotatedAt

	var created *model.Session
	var revoked bool
	sessionRepo := &mockSessionRepo{
		getSessionByRefreshTokenFn: func(ctx context.Context, tokenHash string) (*model.Session, error) {
			return base, nil
		},
		createSessionFn: func(ctx context.Context, s *model.Session) error {
			created = s
			return nil
		},
		deleteByFamilyIDFn: func(ctx context.Context, familyID string) error {
			revoked = true
			return nil
		},
		rotateSessionFn: func(ctx context.Context, currentID string, s *model.Session) error {
			t.Error("grace path must not rotate the parent again")
			return nil
		},
	}

	svc := refreshTestService(sessionRepo)
	_, newRefresh, _, err := svc.RefreshToken(context.Background(), "pl_token", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("expected the concurrent refresh to be served, got: %v", err)
	}
	if revoked {
		t.Error("a refresh inside the grace window must not revoke the family")
	}
	if created == nil || created.FamilyID != "fam-1" {
		t.Fatalf("expected a sibling in family fam-1, got %+v", created)
	}
	if created.RefreshToken != tokenHash(newRefresh) {
		t.Error("expected the returned token to match the created sibling")
	}
}

func TestAuthService_RefreshToken_LosingRaceFallsBackToSibling(t *testing.T) {
	// RotateSession reports that a concurrent request already rotated the row.
	var created *model.Session
	sessionRepo := &mockSessionRepo{
		getSessionByRefreshTokenFn: func(ctx context.Context, tokenHash string) (*model.Session, error) {
			return liveSession("u-1"), nil
		},
		rotateSessionFn: func(ctx context.Context, currentID string, s *model.Session) error {
			return apperrors.ErrSessionAlreadyRotated
		},
		createSessionFn: func(ctx context.Context, s *model.Session) error {
			created = s
			return nil
		},
	}

	svc := refreshTestService(sessionRepo)
	if _, _, _, err := svc.RefreshToken(context.Background(), "pl_token", "127.0.0.1", "test-agent"); err != nil {
		t.Fatalf("expected the losing request to be served, got: %v", err)
	}
	if created == nil {
		t.Fatal("expected a sibling session to be created after losing the race")
	}
}

func TestAuthService_RefreshToken_PastAbsoluteExpiryRevokesFamily(t *testing.T) {
	base := liveSession("u-1")
	base.AbsoluteExpiresAt = time.Now().Add(-time.Minute)

	var revokedFamily string
	sessionRepo := &mockSessionRepo{
		getSessionByRefreshTokenFn: func(ctx context.Context, tokenHash string) (*model.Session, error) {
			return base, nil
		},
		deleteByFamilyIDFn: func(ctx context.Context, familyID string) error {
			revokedFamily = familyID
			return nil
		},
	}

	svc := refreshTestService(sessionRepo)
	_, _, _, err := svc.RefreshToken(context.Background(), "pl_token", "127.0.0.1", "test-agent")
	if !errors.Is(err, apperrors.ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got: %v", err)
	}
	if revokedFamily != "fam-1" {
		t.Errorf("expected family fam-1 to be dropped, got %q", revokedFamily)
	}
}

func TestAuthService_RefreshToken_PastSlidingExpiryDropsOnlyThatToken(t *testing.T) {
	base := liveSession("u-1")
	base.ExpiresAt = time.Now().Add(-time.Minute)

	var deletedToken string
	var familyRevoked bool
	sessionRepo := &mockSessionRepo{
		getSessionByRefreshTokenFn: func(ctx context.Context, tokenHash string) (*model.Session, error) {
			return base, nil
		},
		deleteSessionFn: func(ctx context.Context, hashed string) error {
			deletedToken = hashed
			return nil
		},
		deleteByFamilyIDFn: func(ctx context.Context, familyID string) error {
			familyRevoked = true
			return nil
		},
	}

	svc := refreshTestService(sessionRepo)
	_, _, _, err := svc.RefreshToken(context.Background(), "pl_token", "127.0.0.1", "test-agent")
	if !errors.Is(err, apperrors.ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got: %v", err)
	}
	if deletedToken != tokenHash("pl_token") {
		t.Errorf("expected only the expired token to be deleted, got %q", deletedToken)
	}
	if familyRevoked {
		t.Error("an idle leaf must not take its siblings down with it")
	}
}

func TestAuthService_Login_StartsOwnFamilyWithClientContext(t *testing.T) {
	rawPassword := "ValidPassword123!"
	argonHash, err := pkgpassword.Hash(rawPassword)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	authRepo := &mockAuthRepo{
		findUserByIdentifierFn: func(ctx context.Context, identifier string) (*model.User, error) {
			return &model.User{ID: "u-1", Email: identifier, Password: &argonHash}, nil
		},
	}
	var created *model.Session
	sessionRepo := &mockSessionRepo{
		createSessionFn: func(ctx context.Context, s *model.Session) error {
			created = s
			return nil
		},
	}

	svc := NewAuthService(authRepo, &mockUserRepo{}, sessionRepo, nil, &mockActivityService{}, testAuthConfig(), nil, nil)
	if _, _, _, err := svc.Login(context.Background(), "test@example.com", rawPassword, "203.0.113.7", "test-agent"); err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if created == nil {
		t.Fatal("expected a session to be created")
	}
	if created.FamilyID != created.ID {
		t.Errorf("expected a fresh login to head its own family, got id=%q family=%q", created.ID, created.FamilyID)
	}
	if created.RotatedAt != nil {
		t.Error("expected a fresh session not to be marked rotated")
	}
	if created.IPAddress == nil || *created.IPAddress != "203.0.113.7" {
		t.Error("expected the client IP to be recorded on the session")
	}
	if created.UserAgent == nil || *created.UserAgent != "test-agent" {
		t.Error("expected the user agent to be recorded on the session")
	}
}

func TestAuthService_Logout_RevokesWholeFamily(t *testing.T) {
	var revokedFamily string
	sessionRepo := &mockSessionRepo{
		getSessionByRefreshTokenFn: func(ctx context.Context, tokenHash string) (*model.Session, error) {
			return liveSession("u-1"), nil
		},
		deleteByFamilyIDFn: func(ctx context.Context, familyID string) error {
			revokedFamily = familyID
			return nil
		},
	}

	svc := refreshTestService(sessionRepo)
	if err := svc.Logout(context.Background(), "pl_token"); err != nil {
		t.Fatalf("Logout failed: %v", err)
	}
	if revokedFamily != "fam-1" {
		t.Errorf("expected family fam-1 to be revoked, got %q", revokedFamily)
	}
}

func TestAuthService_Logout_UnknownTokenIsNoOp(t *testing.T) {
	sessionRepo := &mockSessionRepo{
		getSessionByRefreshTokenFn: func(ctx context.Context, tokenHash string) (*model.Session, error) {
			return nil, errors.New("record not found")
		},
		deleteByFamilyIDFn: func(ctx context.Context, familyID string) error {
			t.Error("unknown token must not revoke anything")
			return nil
		},
	}

	svc := refreshTestService(sessionRepo)
	if err := svc.Logout(context.Background(), "pl_unknown"); err != nil {
		t.Fatalf("expected logout to be idempotent, got: %v", err)
	}
}

const testTargetUserID = "0195f3c0-0000-7000-8000-000000000000"

func TestAuthService_RevokeUserSessions(t *testing.T) {
	var revokedFor string
	sessionRepo := &mockSessionRepo{
		deleteByUserIDFn: func(ctx context.Context, userID string) (int64, error) {
			revokedFor = userID
			return 4, nil
		},
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id string, deletedOnly bool) (*model.User, error) {
			return &model.User{ID: id}, nil
		},
	}

	var loggedType string
	var loggedUser *string
	var loggedMeta map[string]any
	activity := &mockActivityService{
		logActivityFn: func(ctx context.Context, userID *string, activityType, status, ip, ua string, errMsg *string, meta map[string]any) {
			loggedType, loggedUser, loggedMeta = activityType, userID, meta
		},
	}

	svc := NewAuthService(&mockAuthRepo{}, userRepo, sessionRepo, nil, activity, testAuthConfig(), nil, nil)
	revoked, err := svc.RevokeUserSessions(context.Background(), testTargetUserID, "admin-1", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("RevokeUserSessions failed: %v", err)
	}

	if revoked != 4 {
		t.Errorf("expected 4 revoked sessions, got %d", revoked)
	}
	if revokedFor != testTargetUserID {
		t.Errorf("expected sessions of %q to be revoked, got %q", testTargetUserID, revokedFor)
	}
	if loggedType != model.ActivitySessionRevoked {
		t.Errorf("expected a %q activity log, got %q", model.ActivitySessionRevoked, loggedType)
	}
	if loggedUser == nil || *loggedUser != testTargetUserID {
		t.Error("expected the activity log to name the affected user")
	}
	if loggedMeta["revoked_by"] != "admin-1" {
		t.Errorf("expected the acting admin in the metadata, got %v", loggedMeta["revoked_by"])
	}
}

func TestAuthService_RevokeUserSessions_RejectsMalformedID(t *testing.T) {
	sessionRepo := &mockSessionRepo{
		deleteByUserIDFn: func(ctx context.Context, userID string) (int64, error) {
			t.Error("a malformed id must not reach the repository")
			return 0, nil
		},
	}

	svc := NewAuthService(&mockAuthRepo{}, &mockUserRepo{}, sessionRepo, nil, &mockActivityService{}, testAuthConfig(), nil, nil)
	_, err := svc.RevokeUserSessions(context.Background(), "not-a-uuid", "admin-1", "127.0.0.1", "test-agent")
	if !errors.Is(err, apperrors.ErrInvalidUserID) {
		t.Fatalf("expected ErrInvalidUserID, got: %v", err)
	}
}

func TestAuthService_RevokeUserSessions_UnknownUser(t *testing.T) {
	// A typo in the id must surface as not-found rather than a success that
	// silently revoked nothing.
	userRepo := &mockUserRepo{
		getByIDFn: func(ctx context.Context, id string, deletedOnly bool) (*model.User, error) {
			return nil, apperrors.ErrUserNotFound
		},
	}
	sessionRepo := &mockSessionRepo{
		deleteByUserIDFn: func(ctx context.Context, userID string) (int64, error) {
			t.Error("an unknown user must not reach the delete")
			return 0, nil
		},
	}

	svc := NewAuthService(&mockAuthRepo{}, userRepo, sessionRepo, nil, &mockActivityService{}, testAuthConfig(), nil, nil)
	_, err := svc.RevokeUserSessions(context.Background(), testTargetUserID, "admin-1", "127.0.0.1", "test-agent")
	if !errors.Is(err, apperrors.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got: %v", err)
	}
}

func TestAuthService_RevokeAllSessions(t *testing.T) {
	called := false
	sessionRepo := &mockSessionRepo{
		deleteAllFn: func(ctx context.Context) (int64, error) {
			called = true
			return 17, nil
		},
	}

	var loggedUser *string
	var loggedMeta map[string]any
	activity := &mockActivityService{
		logActivityFn: func(ctx context.Context, userID *string, activityType, status, ip, ua string, errMsg *string, meta map[string]any) {
			loggedUser, loggedMeta = userID, meta
		},
	}

	svc := NewAuthService(&mockAuthRepo{}, &mockUserRepo{}, sessionRepo, nil, activity, testAuthConfig(), nil, nil)
	revoked, err := svc.RevokeAllSessions(context.Background(), "admin-1", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("RevokeAllSessions failed: %v", err)
	}

	if !called {
		t.Error("expected every session to be deleted")
	}
	if revoked != 17 {
		t.Errorf("expected 17 revoked sessions, got %d", revoked)
	}
	if loggedUser != nil {
		t.Error("a platform-wide revoke belongs to no single account, expected a nil user_id")
	}
	if loggedMeta["scope"] != "all_users" {
		t.Errorf("expected the metadata to mark the scope, got %v", loggedMeta["scope"])
	}
}
