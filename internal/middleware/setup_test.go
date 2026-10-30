package middleware

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"echobackend/config"

	"github.com/labstack/echo/v5"
)

type envelope struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

func testConfig() *config.Config {
	return &config.Config{HTTP: config.HTTPConfig{AllowOrigins: []string{"*"}}}
}

// newTestEcho wires Echo the way cmd/main.go does: enveloped error handler,
// IP extractor and the global middleware chain, plus one route.
func newTestEcho(conf *config.Config) *echo.Echo {
	e := echo.NewWithConfig(echo.Config{
		HTTPErrorHandler: HTTPErrorHandler,
		IPExtractor:      IPExtractor(conf.HTTP.TrustProxy),
	})
	InitMiddleware(e, conf)
	e.GET("/api/ping", func(c *echo.Context) error {
		return c.String(http.StatusOK, c.RealIP())
	})
	e.POST("/api/ping", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	e.GET("/boom", func(*echo.Context) error {
		return io.ErrUnexpectedEOF
	})
	e.GET("/panic", func(*echo.Context) error {
		panic("kaboom")
	})
	return e
}

func serve(e *echo.Echo, req *http.Request) *httptest.ResponseRecorder {
	if req.RemoteAddr == "" {
		req.RemoteAddr = "192.0.2.10:1234"
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) envelope {
	t.Helper()
	if ct := rec.Header().Get(echo.HeaderContentType); !strings.HasPrefix(ct, echo.MIMEApplicationJSON) {
		t.Fatalf("Content-Type = %q, want JSON", ct)
	}
	var body envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	return body
}

func TestUnknownRouteReturnsEnvelope404(t *testing.T) {
	e := newTestEcho(testConfig())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/does-not-exist", nil)
	req.Header.Set(echo.HeaderOrigin, "https://example.com")
	rec := serve(e, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	body := decodeEnvelope(t, rec)
	if body.Success || body.Message != "Route not found" || body.Error != "Resource not found" {
		t.Fatalf("body = %+v", body)
	}
	if rec.Header().Get(echo.HeaderAccessControlAllowOrigin) == "" {
		t.Fatal("404 response is missing CORS headers")
	}
}

func TestWrongMethodReturnsEnvelope405(t *testing.T) {
	e := newTestEcho(testConfig())

	rec := serve(e, httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/ping", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
	body := decodeEnvelope(t, rec)
	if body.Success || body.Message != "Method not allowed" || body.Error != "Method not allowed" {
		t.Fatalf("body = %+v", body)
	}
}

func TestOversizedBodyReturnsEnvelope413(t *testing.T) {
	e := newTestEcho(testConfig())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/ping", strings.NewReader(strings.Repeat("x", 10*1024*1024+1)))
	req.Header.Set(echo.HeaderOrigin, "https://example.com")
	rec := serve(e, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}
	body := decodeEnvelope(t, rec)
	if body.Success || body.Message != "Request Entity Too Large" || body.Error != "Request body too large" {
		t.Fatalf("body = %+v", body)
	}
	if rec.Header().Get(echo.HeaderAccessControlAllowOrigin) == "" {
		t.Fatal("413 response is missing CORS headers")
	}
}

func TestUnhandledErrorsReturnGenericEnvelope500(t *testing.T) {
	e := newTestEcho(testConfig())

	for _, path := range []string{"/boom", "/panic"} {
		rec := serve(e, httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("%s status = %d, want 500", path, rec.Code)
		}
		body := decodeEnvelope(t, rec)
		if body.Success || body.Message != "Internal server error" || body.Error != "" {
			t.Fatalf("%s body = %+v (must not leak the error)", path, body)
		}
	}
}

func TestGlobalRateLimiterTokenBucket(t *testing.T) {
	conf := testConfig()
	conf.HTTP.RateLimitRPS = 1 // burst 2
	e := newTestEcho(conf)

	for i := range 2 {
		rec := serve(e, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ping", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want 200", i+1, rec.Code)
		}
	}

	// Another client keeps its own bucket.
	other := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ping", nil)
	other.RemoteAddr = "192.0.2.99:1234"
	if rec := serve(e, other); rec.Code != http.StatusOK {
		t.Fatalf("other client status = %d, want 200", rec.Code)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ping", nil)
	req.Header.Set(echo.HeaderOrigin, "https://example.com")
	rec := serve(e, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("Retry-After = %q, want 1", got)
	}
	if rec.Header().Get(echo.HeaderAccessControlAllowOrigin) == "" {
		t.Fatal("429 response is missing CORS headers")
	}
	body := decodeEnvelope(t, rec)
	if body.Success || body.Message != "Too many attempts. Please try again later." || body.Error != "Rate limit exceeded" {
		t.Fatalf("body = %+v", body)
	}
}

func TestSecureHeaders(t *testing.T) {
	e := newTestEcho(testConfig())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ping", nil)
	rec := serve(e, req)
	want := map[string]string{
		"X-Xss-Protection":        "1; mode=block",
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "SAMEORIGIN",
		"Content-Security-Policy": "default-src 'self'",
		"Referrer-Policy":         "strict-origin-when-cross-origin",
	}
	for name, value := range want {
		if got := rec.Header().Get(name); got != value {
			t.Errorf("%s = %q, want %q", name, got, value)
		}
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got != "" {
		t.Errorf("plain HTTP HSTS = %q, want none", got)
	}

	req = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ping", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set(echo.HeaderXForwardedProto, "https")
	rec = serve(e, req)
	if got := rec.Header().Get("Strict-Transport-Security"); got != "max-age=3600; includeSubdomains" {
		t.Errorf("forwarded HTTPS HSTS = %q", got)
	}
}

func TestIPExtractorWithTrustProxy(t *testing.T) {
	conf := testConfig()
	conf.HTTP.TrustProxy = true
	e := newTestEcho(conf)

	cases := []struct {
		name string
		peer string
		xff  string
		want string
	}{
		{"no header falls back to peer", "127.0.0.1:1", "", "127.0.0.1"},
		{"untrusted peer ignores header", "198.51.100.5:1", "203.0.113.7", "198.51.100.5"},
		{"single client hop", "127.0.0.1:1", "203.0.113.7", "203.0.113.7"},
		{"rightmost untrusted hop wins over spoofed left entries", "10.0.0.1:1", "198.51.100.1, 203.0.113.7, 10.0.0.2", "203.0.113.7"},
		{"CGNAT hop is a trusted proxy", "100.64.0.1:1", "203.0.113.7, 100.64.1.1", "203.0.113.7"},
		{"all trusted returns leftmost", "127.0.0.1:1", "10.0.0.9, 192.168.1.1", "10.0.0.9"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ping", nil)
			req.RemoteAddr = tc.peer
			if tc.xff != "" {
				req.Header.Set(echo.HeaderXForwardedFor, tc.xff)
			}
			if got := serve(e, req).Body.String(); got != tc.want {
				t.Fatalf("RealIP() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIPExtractorWithoutTrustProxyIgnoresHeader(t *testing.T) {
	e := newTestEcho(testConfig())

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ping", nil)
	req.RemoteAddr = "127.0.0.1:1"
	req.Header.Set(echo.HeaderXForwardedFor, "203.0.113.7")
	if got := serve(e, req).Body.String(); got != "127.0.0.1" {
		t.Fatalf("RealIP() = %q, want peer 127.0.0.1", got)
	}
}
