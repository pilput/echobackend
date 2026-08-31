package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"echobackend/config"
	apperrors "echobackend/internal/apperror"
	"echobackend/internal/model"
	"echobackend/internal/service"
	"echobackend/pkg/response"
	"echobackend/pkg/validator"

	"github.com/labstack/echo/v5"
)

var (
	_ service.AuthService         = (*mockAuthService)(nil)
	_ service.AuthActivityService = (*mockAuthActivityService)(nil)
)

type mockAuthService struct {
	loginFn             func(ctx context.Context, identifier, password, ipAddress, userAgent string) (string, string, *model.User, error)
	getGithubOAuthURLFn func(state string) string
}

func (m *mockAuthService) Register(ctx context.Context, email, username, password string) (*model.User, error) {
	return nil, nil
}

func (m *mockAuthService) Login(ctx context.Context, identifier, password, ipAddress, userAgent string) (string, string, *model.User, error) {
	if m.loginFn != nil {
		return m.loginFn(ctx, identifier, password, ipAddress, userAgent)
	}
	return "", "", nil, nil
}

func (m *mockAuthService) ForgotPassword(ctx context.Context, email, ipAddress, userAgent string) error {
	return nil
}

func (m *mockAuthService) ResetPassword(ctx context.Context, token, password, ipAddress, userAgent string) error {
	return nil
}

func (m *mockAuthService) RefreshToken(ctx context.Context, refreshToken, ipAddress, userAgent string) (string, string, *model.User, error) {
	return "", "", nil, nil
}

func (m *mockAuthService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword, ipAddress, userAgent string) error {
	return nil
}

func (m *mockAuthService) Logout(ctx context.Context, refreshToken string) error {
	return nil
}

func (m *mockAuthService) GetProfile(ctx context.Context, userID string) (*model.User, error) {
	return nil, nil
}

func (m *mockAuthService) GetGithubOAuthURL(state string) string {
	if m.getGithubOAuthURLFn != nil {
		return m.getGithubOAuthURLFn(state)
	}
	return "https://github.com/login/oauth/authorize?state=" + state
}

func (m *mockAuthService) GetGithubToken(_ context.Context, code string) (string, error) {
	return "", nil
}

func (m *mockAuthService) SignInWithGithub(ctx context.Context, githubUser *service.GithubUser, ipAddress, userAgent string) (string, string, *model.User, error) {
	return "", "", nil, nil
}

func (m *mockAuthService) CreateOAuthExchangeCode(ctx context.Context, accessToken, refreshToken string, user *model.User) (string, error) {
	return "", nil
}

func (m *mockAuthService) ExchangeOAuthCode(ctx context.Context, code string) (string, string, *model.User, error) {
	return "", "", nil, nil
}

func (m *mockAuthService) UpdateProfile(ctx context.Context, userID, username, firstName, lastName string) (*model.User, error) {
	return nil, nil
}

func (m *mockAuthService) DeleteAccount(ctx context.Context, userID string) error {
	return nil
}

func (m *mockAuthService) CheckUsernameExists(ctx context.Context, username string) (bool, error) {
	return false, nil
}

type mockAuthActivityService struct{}

func (m *mockAuthActivityService) LogActivity(ctx context.Context, userID *string, activityType, status, ipAddress, userAgent string, errorMessage *string, metadata map[string]any) {
}

func (m *mockAuthActivityService) GetActivityLogs(ctx context.Context, userID, activityType string, limit, offset int) ([]*model.AuthActivityLog, int64, error) {
	return nil, 0, nil
}

func (m *mockAuthActivityService) GetRecentActivity(ctx context.Context, userID string, limit int) ([]*model.AuthActivityLog, error) {
	return nil, nil
}

func (m *mockAuthActivityService) GetFailedLogins(ctx context.Context, since time.Time, limit, offset int) ([]*model.AuthActivityLog, int64, error) {
	return nil, 0, nil
}

func newAuthTestContext(t *testing.T, method, target, body string) (*echo.Context, *httptest.ResponseRecorder) {
	t.Helper()

	e := echo.New()
	e.Validator = validator.NewValidator()

	req := httptest.NewRequestWithContext(context.Background(), method, target, bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return echo.NewContext(req, rec, e), rec
}

func decodeAuthResponse(t *testing.T, rec *httptest.ResponseRecorder) response.APIResponse {
	t.Helper()

	var out response.APIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode response: %v\nbody: %s", err, rec.Body.String())
	}
	return out
}

func TestAuthHandlerLoginSuccess(t *testing.T) {
	username := "cecep"
	h := NewAuthHandler(&mockAuthService{
		loginFn: func(ctx context.Context, identifier, password, ipAddress, userAgent string) (string, string, *model.User, error) {
			if identifier != "cecep" || password != "secret123" {
				t.Fatalf("unexpected credentials %q/%q", identifier, password)
			}
			return "access-token", "refresh-token", &model.User{
				ID:       "user-1",
				Email:    "cecep@example.com",
				Username: &username,
			}, nil
		},
	}, &mockAuthActivityService{}, config.FrontendConfig{})

	c, rec := newAuthTestContext(t, http.MethodPost, "/api/auth/login", `{"identifier":"cecep","password":"secret123"}`)

	if err := h.Login(c); err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	out := decodeAuthResponse(t, rec)
	if !out.Success || out.Message != "Login successful" {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestAuthHandlerLoginInvalidCredentials(t *testing.T) {
	h := NewAuthHandler(&mockAuthService{
		loginFn: func(ctx context.Context, identifier, password, ipAddress, userAgent string) (string, string, *model.User, error) {
			return "", "", nil, apperrors.ErrInvalidCredentials
		},
	}, &mockAuthActivityService{}, config.FrontendConfig{})

	c, rec := newAuthTestContext(t, http.MethodPost, "/api/auth/login", `{"identifier":"cecep","password":"secret123"}`)

	if err := h.Login(c); err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	out := decodeAuthResponse(t, rec)
	if out.Success || out.Message != "Invalid identifier or password" {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestAuthHandlerGetProfileRequiresUser(t *testing.T) {
	h := NewAuthHandler(&mockAuthService{}, &mockAuthActivityService{}, config.FrontendConfig{})
	c, rec := newAuthTestContext(t, http.MethodGet, "/api/auth/profile", "")

	if err := h.GetProfile(c); err != nil {
		t.Fatalf("GetProfile returned error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAuthHandlerGithubOAuthRedirectSetsStateCookie(t *testing.T) {
	var stateFromService string
	h := NewAuthHandler(&mockAuthService{
		getGithubOAuthURLFn: func(state string) string {
			stateFromService = state
			return "https://github.com/login/oauth/authorize?state=" + state
		},
	}, &mockAuthActivityService{}, config.FrontendConfig{URL: "https://pilput.net"})

	c, rec := newAuthTestContext(t, http.MethodGet, "/api/auth/oauth/github", "")

	if err := h.GithubOAuthRedirect(c); err != nil {
		t.Fatalf("GithubOAuthRedirect returned error: %v", err)
	}

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if stateFromService == "" {
		t.Fatal("expected generated OAuth state to be passed to auth service")
	}
	if location := rec.Header().Get(echo.HeaderLocation); !strings.Contains(location, stateFromService) {
		t.Fatalf("redirect location %q does not contain state %q", location, stateFromService)
	}
	if cookie := rec.Result().Cookies()[0]; cookie.Name != "github_oauth_state" || cookie.Value != stateFromService || !cookie.HttpOnly || !cookie.Secure {
		t.Fatalf("unexpected oauth cookie: %+v", cookie)
	}
}

func TestAppendQueryParamPreservesExistingQuery(t *testing.T) {
	got := appendQueryParam("https://pilput.net/auth/callback?from=github", "code", "oc_123")

	if !strings.Contains(got, "from=github") || !strings.Contains(got, "code=oc_123") {
		t.Fatalf("query params not preserved, got %q", got)
	}
}
