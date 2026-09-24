package handler

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/platform/realtime"
	"echobackend/internal/service"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

// stubGuildChannelService implements only what the stream handler uses.
type stubGuildChannelService struct {
	service.GuildChannelService
	hub      *realtime.Hub
	isMember bool
}

func (s *stubGuildChannelService) Subscribe(_ context.Context, guildSlug, _ string) (string, *realtime.Subscription, error) {
	if guildSlug != "g" {
		return "", nil, apperrors.ErrGuildNotFound
	}
	return "guild-1", s.hub.Subscribe(service.GuildEventTopic("guild-1")), nil
}

func (s *stubGuildChannelService) IsMember(context.Context, string, string) (bool, error) {
	return s.isMember, nil
}

func newStreamServer(t *testing.T, svc service.GuildChannelService) *httptest.Server {
	t.Helper()
	e := echo.New()
	h := NewGuildChannelHandler(svc)
	withUser := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("user", jwt.MapClaims{"user_id": "user-1"})
			return next(c)
		}
	}
	e.GET("/api/guilds/:slug/events", h.StreamEvents, withUser)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	return srv
}

func TestStreamEvents_ForwardsEventsUntilHubCloses(t *testing.T) {
	hub := realtime.NewHub(nil)
	hub.Start()
	srv := newStreamServer(t, &stubGuildChannelService{hub: hub, isMember: true})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/guilds/g/events", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET events: %v", err)
	}
	defer res.Body.Close()

	if ct := res.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q", ct)
	}
	reader := bufio.NewReader(res.Body)
	readLine := func() string {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read stream: %v", err)
		}
		return strings.TrimRight(line, "\n")
	}

	if line := readLine(); line != ": connected" {
		t.Fatalf("first line = %q", line)
	}
	readLine() // blank frame separator

	// The subscription is registered before the handler writes ": connected".
	_ = hub.Publish(ctx, service.GuildEventTopic("guild-1"), dto.GuildEvent{Type: dto.GuildEventMessageCreated, GuildID: "guild-1"})
	if line := readLine(); !strings.HasPrefix(line, "data: ") || !strings.Contains(line, `"type":"message.created"`) {
		t.Fatalf("event line = %q", line)
	}
	readLine()

	// Shutting the hub down must end the stream so server shutdown is not held.
	_ = hub.Close()
	if _, err := reader.ReadString('\n'); err == nil {
		t.Fatal("stream should end after the hub closes")
	}
}

func TestStreamEvents_HiddenGuildIsNotFound(t *testing.T) {
	hub := realtime.NewHub(nil)
	hub.Start()
	defer func() { _ = hub.Close() }()
	srv := newStreamServer(t, &stubGuildChannelService{hub: hub, isMember: true})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/api/guilds/other/events", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET events: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
}

func TestHandleGuildChannelError_NonMemberIsForbidden(t *testing.T) {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil), rec)

	_ = handleGuildChannelError(c, "x", apperrors.ErrNotGuildMember)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}
