package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/service"

	"github.com/labstack/echo/v5"
)

var _ service.NotificationService = (*mockNotificationService)(nil)

type mockNotificationService struct {
	markAsReadFn func(ctx context.Context, id, userID string) (*dto.NotificationResponse, error)
	called       bool
}

func (m *mockNotificationService) CreateNotification(ctx context.Context, req *dto.CreateNotificationRequest) (*dto.NotificationResponse, error) {
	m.called = true
	return nil, nil
}

func (m *mockNotificationService) GetNotifications(ctx context.Context, userID string, filter *dto.NotificationListFilter) ([]*dto.NotificationResponse, int64, error) {
	m.called = true
	return nil, 0, errors.New("db down")
}

func (m *mockNotificationService) GetUnreadCount(ctx context.Context, userID string) (*dto.NotificationUnreadCountResponse, error) {
	m.called = true
	return nil, errors.New("db down")
}

func (m *mockNotificationService) MarkAsRead(ctx context.Context, id, userID string) (*dto.NotificationResponse, error) {
	m.called = true
	if m.markAsReadFn != nil {
		return m.markAsReadFn(ctx, id, userID)
	}
	return &dto.NotificationResponse{ID: id}, nil
}

func (m *mockNotificationService) MarkAllAsRead(ctx context.Context, userID string) (*dto.NotificationMarkAllReadResponse, error) {
	m.called = true
	return nil, errors.New("db down")
}

func registerNotificationRoutes(e *echo.Echo, h *NotificationHandler) {
	e.GET("/api/notifications", h.GetNotifications)
	e.GET("/api/notifications/unread-count", h.GetUnreadCount)
	e.PATCH("/api/notifications/read-all", h.MarkAllAsRead)
	e.PATCH("/api/notifications/:id/read", h.MarkAsRead)
}

func serveNotificationRoute(t *testing.T, svc service.NotificationService, method, target string) (int, string, bool) {
	t.Helper()
	h := NewNotificationHandler(svc)
	rec := serveAuthRoute(t, func(e *echo.Echo, _ *AuthHandler) { registerNotificationRoutes(e, h) }, nil, method, target, "", "user-1")
	out := decodeAuthResponse(t, rec)
	return rec.Code, out.Message, out.Success
}

const validNotificationID = "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"

func TestNotificationMarkAsReadRejectsMalformedID(t *testing.T) {
	svc := &mockNotificationService{}
	status, msg, _ := serveNotificationRoute(t, svc, http.MethodPatch, "/api/notifications/abc/read")
	if status != http.StatusBadRequest || msg != MsgInvalidNotificationID {
		t.Fatalf("got %d %q, want 400 %q", status, msg, MsgInvalidNotificationID)
	}
	if svc.called {
		t.Fatal("service must not be called for a malformed id")
	}
}

func TestNotificationMarkAsReadNotFoundIs404(t *testing.T) {
	svc := &mockNotificationService{
		markAsReadFn: func(ctx context.Context, id, userID string) (*dto.NotificationResponse, error) {
			return nil, apperrors.ErrNotificationNotFound
		},
	}
	status, msg, _ := serveNotificationRoute(t, svc, http.MethodPatch, "/api/notifications/"+validNotificationID+"/read")
	if status != http.StatusNotFound || msg != "Failed to mark notification as read" {
		t.Fatalf("got %d %q, want 404", status, msg)
	}
}

func TestNotificationMarkAsReadSuccess(t *testing.T) {
	status, _, ok := serveNotificationRoute(t, &mockNotificationService{}, http.MethodPatch, "/api/notifications/"+validNotificationID+"/read")
	if status != http.StatusOK || !ok {
		t.Fatalf("got %d success=%v, want 200", status, ok)
	}
}

func TestNotificationHandlersMapUnknownErrorsTo500(t *testing.T) {
	cases := []struct{ method, target, msg string }{
		{http.MethodGet, "/api/notifications", "Failed to get notifications"},
		{http.MethodGet, "/api/notifications/unread-count", "Failed to get unread notification count"},
		{http.MethodPatch, "/api/notifications/read-all", "Failed to mark all notifications as read"},
	}
	for _, tc := range cases {
		status, msg, _ := serveNotificationRoute(t, &mockNotificationService{}, tc.method, tc.target)
		if status != http.StatusInternalServerError || msg != tc.msg {
			t.Errorf("%s %s: got %d %q, want 500 %q", tc.method, tc.target, status, msg, tc.msg)
		}
	}
}
