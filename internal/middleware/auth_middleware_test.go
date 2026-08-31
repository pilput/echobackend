package middleware

import (
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"echobackend/config"
	"echobackend/internal/dto"
	"echobackend/pkg/response"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

type mockUserService struct {
	getAdminByIDFn func(ctx context.Context, id string, deletedOnly bool) (*dto.UserResponse, error)
}

func (m *mockUserService) GetByID(ctx context.Context, id string) (*dto.UserResponse, error) {
	return nil, nil
}

func (m *mockUserService) GetAdminByID(ctx context.Context, id string, deletedOnly bool) (*dto.UserResponse, error) {
	if m.getAdminByIDFn != nil {
		return m.getAdminByIDFn(ctx, id, deletedOnly)
	}
	return nil, nil
}

func (m *mockUserService) GetMe(ctx context.Context, id string) (*dto.CurrentUserResponse, error) {
	return nil, nil
}

func (m *mockUserService) GetByUsername(ctx context.Context, username string) (*dto.UserResponse, error) {
	return nil, nil
}

func (m *mockUserService) GetUsers(ctx context.Context, offset int, limit int, deletedFilter dto.UserDeletedFilter) ([]*dto.UserResponse, int64, error) {
	return nil, 0, nil
}

func (m *mockUserService) Delete(ctx context.Context, id string) error { return nil }

func (m *mockUserService) Restore(ctx context.Context, id string) (*dto.UserResponse, error) {
	return nil, nil
}

func (m *mockUserService) UploadAvatar(ctx context.Context, userID string, file *multipart.FileHeader) (string, error) {
	return "", nil
}

func (m *mockUserService) CreateUser(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error) {
	return nil, nil
}

func (m *mockUserService) UpdateUser(ctx context.Context, id string, req *dto.UpdateUserRequest) (*dto.UserResponse, error) {
	return nil, nil
}

func newAuthMiddlewareForTest(secret string, users *mockUserService) *AuthMiddleware {
	return NewAuthMiddleware(&config.Config{
		Auth: config.AuthConfig{JWTSecret: secret},
	}, users)
}

func signTestToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	if _, ok := claims["exp"]; !ok {
		claims["exp"] = time.Now().Add(time.Hour).Unix()
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func decodeAPIResponse(t *testing.T, rec *httptest.ResponseRecorder) response.APIResponse {
	t.Helper()
	var body response.APIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v body=%q", err, rec.Body.String())
	}
	return body
}

func TestAuth_MissingHeader(t *testing.T) {
	e := echo.New()
	mw := newAuthMiddlewareForTest("test-secret", &mockUserService{})
	handler := mw.Auth()(func(c *echo.Context) error {
		t.Fatal("next should not run")
		return nil
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	body := decodeAPIResponse(t, rec)
	if body.Success {
		t.Fatal("expected success=false")
	}
	if body.Message != "Missing authorization header" {
		t.Fatalf("message = %q", body.Message)
	}
	if body.Error != "Unauthorized access" {
		t.Fatalf("error field = %q, want standard Unauthorized access", body.Error)
	}
}

func TestAuth_MalformedHeader(t *testing.T) {
	e := echo.New()
	mw := newAuthMiddlewareForTest("test-secret", &mockUserService{})
	handler := mw.Auth()(func(c *echo.Context) error {
		t.Fatal("next should not run")
		return nil
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Token abc")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	body := decodeAPIResponse(t, rec)
	if body.Message != "Invalid authorization header" {
		t.Fatalf("message = %q", body.Message)
	}
	// Must not leak internal parse details
	if strings.Contains(strings.ToLower(body.Message), "bearer") {
		t.Fatalf("message leaked format detail: %q", body.Message)
	}
}

func TestAuth_InvalidTokenDoesNotLeakDetails(t *testing.T) {
	e := echo.New()
	mw := newAuthMiddlewareForTest("test-secret", &mockUserService{})
	handler := mw.Auth()(func(c *echo.Context) error {
		t.Fatal("next should not run")
		return nil
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-jwt")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	body := decodeAPIResponse(t, rec)
	if body.Message != "Invalid or expired token" {
		t.Fatalf("message = %q", body.Message)
	}
	raw := rec.Body.String()
	for _, leak := range []string{"token parsing failed", "signature is invalid", "unexpected signing method", "jwt:"} {
		if strings.Contains(strings.ToLower(raw), strings.ToLower(leak)) {
			t.Fatalf("response leaked token details (%q): %s", leak, raw)
		}
	}
}

func TestAuth_ValidToken(t *testing.T) {
	const secret = "test-secret"
	e := echo.New()
	mw := newAuthMiddlewareForTest(secret, &mockUserService{})

	var gotUserID string
	handler := mw.Auth()(func(c *echo.Context) error {
		claims, ok := c.Get("user").(jwt.MapClaims)
		if !ok {
			t.Fatal("user claims not set")
		}
		gotUserID, _ = claims["user_id"].(string)
		return c.NoContent(http.StatusNoContent)
	})

	token := signTestToken(t, secret, jwt.MapClaims{"user_id": "user-1"})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if gotUserID != "user-1" {
		t.Fatalf("user_id = %q, want user-1", gotUserID)
	}
}

func TestAuthAdmin_ForbiddenUsesStandardResponse(t *testing.T) {
	const secret = "test-secret"
	falseVal := false
	users := &mockUserService{
		getAdminByIDFn: func(ctx context.Context, id string, deletedOnly bool) (*dto.UserResponse, error) {
			return &dto.UserResponse{ID: id, IsSuperAdmin: &falseVal}, nil
		},
	}
	mw := newAuthMiddlewareForTest(secret, users)

	e := echo.New()
	handler := mw.AuthAdmin()(func(c *echo.Context) error {
		t.Fatal("next should not run")
		return nil
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user", jwt.MapClaims{"user_id": "user-1"})

	if err := handler(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}

	body := decodeAPIResponse(t, rec)
	if body.Message != "Insufficient privileges" {
		t.Fatalf("message = %q", body.Message)
	}
	if body.Error != "Access forbidden" {
		t.Fatalf("error field = %q", body.Error)
	}
}

func TestAuthAdmin_AllowsSuperAdmin(t *testing.T) {
	trueVal := true
	users := &mockUserService{
		getAdminByIDFn: func(ctx context.Context, id string, deletedOnly bool) (*dto.UserResponse, error) {
			return &dto.UserResponse{ID: id, IsSuperAdmin: &trueVal}, nil
		},
	}
	mw := newAuthMiddlewareForTest("test-secret", users)

	e := echo.New()
	called := false
	handler := mw.AuthAdmin()(func(c *echo.Context) error {
		called = true
		return c.NoContent(http.StatusNoContent)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user", jwt.MapClaims{"user_id": "admin-1"})

	if err := handler(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !called {
		t.Fatal("next was not called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestAuthAdmin_AllowsSuperAdminFromClaimFastPath(t *testing.T) {
	// mockUserService has no getAdminByIDFn defined, so if called, it would panic.
	users := &mockUserService{}
	mw := newAuthMiddlewareForTest("test-secret", users)

	e := echo.New()
	called := false
	handler := mw.AuthAdmin()(func(c *echo.Context) error {
		called = true
		return c.NoContent(http.StatusNoContent)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user", jwt.MapClaims{"user_id": "admin-1", "is_super_admin": true})

	if err := handler(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !called {
		t.Fatal("next was not called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}
