package di

import (
	"context"

	"echobackend/internal/dto"
	"echobackend/internal/modules/blog"
	"echobackend/internal/service"
)

// The adapters below let modules use core services through small interfaces
// they own, so modules never import the legacy service/dto packages.

// adminChecker implements blog.AdminChecker on top of the user service. It
// always reads the database: the is_super_admin JWT claim stays in the token
// until it expires, so trusting it would keep a demoted admin privileged.
type adminChecker struct {
	users service.UserService
}

func (a adminChecker) IsSuperAdmin(ctx context.Context, userID string) bool {
	if a.users == nil || userID == "" {
		return false
	}
	user, err := a.users.GetAdminByID(ctx, userID, false)
	if err != nil || user.IsSuperAdmin == nil {
		return false
	}
	return *user.IsSuperAdmin
}

// notifier implements blog.Notifier on top of the notification service.
type notifier struct {
	notifications service.NotificationService
}

func (n notifier) Notify(ctx context.Context, msg *blog.Notification) error {
	_, err := n.notifications.CreateNotification(ctx, &dto.CreateNotificationRequest{
		UserID:  msg.UserID,
		Type:    msg.Type,
		Title:   msg.Title,
		Message: msg.Message,
		Data:    msg.Data,
	})
	return err
}
