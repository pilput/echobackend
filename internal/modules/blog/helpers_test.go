package blog

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"echobackend/pkg/response"
	"echobackend/pkg/validator"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

// serveRoute registers routes and drives one request through the router,
// which is the only way a handler reading c.Param sees its path parameters.
// actorID, when non-empty, stands in for what the auth middleware would set.
func serveRoute(t *testing.T, register func(e *echo.Echo), method, target, body, actorID string) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	e.Validator = validator.NewValidator()
	if actorID != "" {
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
				c.Set("user", jwt.MapClaims{"user_id": actorID})
				return next(c)
			}
		})
	}
	register(e)

	req := httptest.NewRequestWithContext(context.Background(), method, target, bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// decodeResponse decodes the standard JSON envelope or fails the test.
func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) response.APIResponse {
	t.Helper()

	var out response.APIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode response: %v\nbody: %s", err, rec.Body.String())
	}
	return out
}
