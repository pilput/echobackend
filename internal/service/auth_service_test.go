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
	deleteSessionFn            func(ctx context.Context, tokenHash string) error
	deleteByUserIDFn           func(ctx context.Context, userID string) error
	cleanExpiredSessionsFn     func(ctx context.Context) error
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

func (m *mockSessionRepo) DeleteByUserID(ctx context.Context, userID string) error {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(ctx, userID)
	}
	return nil
}

func (m *mockSessionRepo) UpdateSession(ctx context.Context, s *model.Session) error {
	return nil
}

func (m *mockSessionRepo) CleanExpiredSessions(ctx context.Context) error {
	if m.cleanExpiredSessionsFn != nil {
		return m.cleanExpiredSessionsFn(ctx)
	}
	return nil
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
			JWTSecret:          "01234567890123456789012345678901", // >= 32 chars
			JWTExpiry:          15 * time.Minute,
			RefreshTokenExpiry: 7 * 24 * time.Hour,
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
