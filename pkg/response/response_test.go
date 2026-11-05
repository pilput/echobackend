package response

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pkgvalidator "echobackend/pkg/validator"

	"github.com/labstack/echo/v5"
)

func newCtx(t *testing.T) (*echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := echo.NewContext(req, rec, e)
	return c, rec
}

func decode(t *testing.T, body []byte) APIResponse {
	t.Helper()
	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode response: %v\nbody=%s", err, string(body))
	}
	return resp
}

func TestSuccess(t *testing.T) {
	c, rec := newCtx(t)
	if err := Success(c, "ok", map[string]string{"hello": "world"}); err != nil {
		t.Fatalf("Success returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := decode(t, rec.Body.Bytes())
	if !body.Success {
		t.Errorf("expected Success=true, got false")
	}
	if body.Message != "ok" {
		t.Errorf("Message = %q, want %q", body.Message, "ok")
	}
	if body.Data == nil {
		t.Errorf("expected Data populated")
	}
}

func TestSuccessWithMeta(t *testing.T) {
	c, rec := newCtx(t)
	meta := PaginationMeta{TotalItems: 5, Offset: 0, Limit: 10, TotalPages: 1}
	if err := SuccessWithMeta(c, "ok", []int{1, 2, 3}, meta); err != nil {
		t.Fatalf("SuccessWithMeta returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := decode(t, rec.Body.Bytes())
	if body.Meta == nil {
		t.Fatalf("expected meta in response")
	}
}

func TestCreated(t *testing.T) {
	c, rec := newCtx(t)
	if err := Created(c, "created", nil); err != nil {
		t.Fatalf("Created returned error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestBadRequest(t *testing.T) {
	c, rec := newCtx(t)
	if err := BadRequest(c, "bad input", errors.New("missing field")); err != nil {
		t.Fatalf("BadRequest returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	body := decode(t, rec.Body.Bytes())
	if body.Success {
		t.Errorf("expected Success=false")
	}
	if body.Error != "missing field" {
		t.Errorf("expected error %q, got %q", "missing field", body.Error)
	}
}

func TestBadRequest_NilError(t *testing.T) {
	c, rec := newCtx(t)
	_ = BadRequest(c, "bad", nil)
	body := decode(t, rec.Body.Bytes())
	if body.Error != "" {
		t.Errorf("expected empty error string when err=nil, got %q", body.Error)
	}
}

func TestUnauthorized(t *testing.T) {
	c, rec := newCtx(t)
	_ = Unauthorized(c, "no token")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	body := decode(t, rec.Body.Bytes())
	if body.Error != "Unauthorized access" {
		t.Errorf("Error = %q", body.Error)
	}
}

func TestForbidden(t *testing.T) {
	c, rec := newCtx(t)
	_ = Forbidden(c, "no rights")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestServiceUnavailable(t *testing.T) {
	c, rec := newCtx(t)
	_ = ServiceUnavailable(c, "AI article generation is not available")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", rec.Code)
	}
	got := decode(t, rec.Body.Bytes())
	if got.Success || got.Message != "AI article generation is not available" || got.Error != "Service Unavailable" {
		t.Fatalf("body = %+v", got)
	}
}

func TestNotFound(t *testing.T) {
	c, rec := newCtx(t)
	_ = NotFound(c, "missing", errors.New("post not found"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	body := decode(t, rec.Body.Bytes())
	if body.Error != "post not found" {
		t.Errorf("Error = %q", body.Error)
	}
}

func TestNotFound_NilError(t *testing.T) {
	c, rec := newCtx(t)
	_ = NotFound(c, "missing", nil)
	body := decode(t, rec.Body.Bytes())
	if body.Error != "Resource not found" {
		t.Errorf("expected default error message, got %q", body.Error)
	}
}

// TestInternalServerError_DoesNotLeakError is the documented contract:
// the raw error string MUST NOT appear in the response body — only logged server-side.
func TestInternalServerError_DoesNotLeakError(t *testing.T) {
	c, rec := newCtx(t)
	secret := "DSN=postgres://user:hunter2@host/db"
	_ = InternalServerError(c, "boom", errors.New(secret))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), secret) {
		t.Fatalf("InternalServerError leaked raw error: %s", rec.Body.String())
	}
	body := decode(t, rec.Body.Bytes())
	if body.Success {
		t.Errorf("expected Success=false")
	}
	if body.Message != "boom" {
		t.Errorf("Message = %q", body.Message)
	}
}

func TestValidationError(t *testing.T) {
	c, rec := newCtx(t)
	_ = ValidationError(c, "invalid", errors.New("field x"))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
}

func TestFromValidateError_Structured(t *testing.T) {
	c, rec := newCtx(t)
	verr := pkgvalidator.ValidationErrors{
		Errors: []pkgvalidator.ValidationError{
			{Field: "Email", Message: "Email must be a valid email address", Tag: "email"},
		},
	}
	_ = FromValidateError(c, verr)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", rec.Code)
	}
	body := decode(t, rec.Body.Bytes())
	if body.Success {
		t.Errorf("expected Success=false")
	}
	if body.Errors == nil {
		t.Errorf("expected Errors field populated")
	}
}

func TestFromValidateError_Generic(t *testing.T) {
	c, rec := newCtx(t)
	_ = FromValidateError(c, errors.New("plain error"))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestConflict(t *testing.T) {
	c, rec := newCtx(t)
	_ = Conflict(c, "duplicate", "user already exists")
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d", rec.Code)
	}
	body := decode(t, rec.Body.Bytes())
	if body.Error != "user already exists" {
		t.Errorf("Error = %q", body.Error)
	}
}

func TestCalculatePaginationMeta(t *testing.T) {
	tests := []struct {
		name       string
		total      int64
		offset     int
		limit      int
		wantPages  int
		wantLimit  int
		wantOffset int
	}{
		{"exact pages", 100, 0, 10, 10, 10, 0},
		{"with remainder", 95, 0, 10, 10, 10, 0},
		{"single page", 5, 0, 10, 1, 10, 0},
		{"zero items", 0, 0, 10, 0, 10, 0},
		{"limit zero -> default 10", 25, 0, 0, 3, 10, 0},
		{"limit negative -> default 10", 25, 0, -5, 3, 10, 0},
		{"offset preserved", 30, 20, 5, 6, 5, 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := CalculatePaginationMeta(tt.total, tt.offset, tt.limit)
			if meta.TotalPages != tt.wantPages {
				t.Errorf("TotalPages = %d, want %d", meta.TotalPages, tt.wantPages)
			}
			if meta.Limit != tt.wantLimit {
				t.Errorf("Limit = %d, want %d", meta.Limit, tt.wantLimit)
			}
			if meta.Offset != tt.wantOffset {
				t.Errorf("Offset = %d, want %d", meta.Offset, tt.wantOffset)
			}
			if int64(meta.TotalItems) != tt.total {
				t.Errorf("TotalItems = %d, want %d", meta.TotalItems, tt.total)
			}
		})
	}
}
