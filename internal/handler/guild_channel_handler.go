package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/service"
	"echobackend/pkg/applog"
	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

var guildChatLog = applog.Component("guild_chat")

const (
	// guildStreamHeartbeat keeps proxies from closing an idle stream and is
	// also how often a subscriber's membership is re-checked, so a member who
	// leaves stops receiving events within this interval.
	guildStreamHeartbeat = 25 * time.Second
	// guildStreamWriteTimeout bounds each write to the stream. The server-wide
	// WriteTimeout (60s) would otherwise cut every stream after a minute.
	guildStreamWriteTimeout = 10 * time.Second
)

type GuildChannelHandler struct {
	channelService service.GuildChannelService
}

func NewGuildChannelHandler(channelService service.GuildChannelService) *GuildChannelHandler {
	return &GuildChannelHandler{channelService: channelService}
}

func (h *GuildChannelHandler) ListChannels(c *echo.Context) error {
	guildSlug := c.Param("slug")
	viewerID, _ := GetUserIDFromClaims(c)

	channels, err := h.channelService.ListChannels(c.Request().Context(), guildSlug, viewerID)
	if err != nil {
		return handleGuildChannelError(c, "Failed to fetch channels", err)
	}
	return response.Success(c, "Channels fetched successfully", channels)
}

func (h *GuildChannelHandler) CreateChannel(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}

	var req dto.CreateGuildChannelRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request body", err)
	}
	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	channel, err := h.channelService.CreateChannel(c.Request().Context(), c.Param("slug"), userID, &req)
	if err != nil {
		return handleGuildChannelError(c, "Failed to create channel", err)
	}
	return response.Created(c, "Channel created successfully", channel)
}

func (h *GuildChannelHandler) UpdateChannel(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}

	var req dto.UpdateGuildChannelRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request body", err)
	}
	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	channel, err := h.channelService.UpdateChannel(c.Request().Context(), c.Param("slug"), c.Param("channelId"), userID, &req)
	if err != nil {
		return handleGuildChannelError(c, "Failed to update channel", err)
	}
	return response.Success(c, "Channel updated successfully", channel)
}

func (h *GuildChannelHandler) DeleteChannel(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}

	if err := h.channelService.DeleteChannel(c.Request().Context(), c.Param("slug"), c.Param("channelId"), userID); err != nil {
		return handleGuildChannelError(c, "Failed to delete channel", err)
	}
	return response.Success(c, "Channel deleted successfully", nil)
}

// ListMessages returns channel history newest first, paginated backwards with
// the ?before=<message id> cursor rather than offset, so that messages
// arriving meanwhile do not shift the pages.
func (h *GuildChannelHandler) ListMessages(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}
	limit, _ := ParsePaginationParams(c, 50)

	messages, meta, err := h.channelService.ListMessages(
		c.Request().Context(), c.Param("slug"), c.Param("channelId"), userID, c.QueryParam("before"), limit)
	if err != nil {
		return handleGuildChannelError(c, "Failed to fetch messages", err)
	}
	return response.SuccessWithMeta(c, "Messages fetched successfully", messages, meta)
}

func (h *GuildChannelHandler) SendMessage(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}

	var req dto.CreateGuildMessageRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request body", err)
	}
	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	message, err := h.channelService.SendMessage(c.Request().Context(), c.Param("slug"), c.Param("channelId"), userID, &req)
	if err != nil {
		return handleGuildChannelError(c, "Failed to send message", err)
	}
	return response.Created(c, "Message sent successfully", message)
}

func (h *GuildChannelHandler) EditMessage(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}

	var req dto.UpdateGuildMessageRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "Invalid request body", err)
	}
	if err := c.Validate(req); err != nil {
		return response.FromValidateError(c, err)
	}

	message, err := h.channelService.EditMessage(
		c.Request().Context(), c.Param("slug"), c.Param("channelId"), c.Param("messageId"), userID, &req)
	if err != nil {
		return handleGuildChannelError(c, "Failed to edit message", err)
	}
	return response.Success(c, "Message updated successfully", message)
}

func (h *GuildChannelHandler) DeleteMessage(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}

	err := h.channelService.DeleteMessage(
		c.Request().Context(), c.Param("slug"), c.Param("channelId"), c.Param("messageId"), userID)
	if err != nil {
		return handleGuildChannelError(c, "Failed to delete message", err)
	}
	return response.Success(c, "Message deleted successfully", nil)
}

// StreamEvents holds a Server-Sent Events stream open and forwards every
// realtime event of the guild (all channels) to a member. Each event is one
// `data:` line holding a dto.GuildEvent. The stream ends when the client
// disconnects, the member leaves the guild, the subscriber falls too far
// behind, or the server shuts down; clients should reconnect and refetch
// recent history to fill any gap.
func (h *GuildChannelHandler) StreamEvents(c *echo.Context) error {
	userID, ok := GetUserIDFromClaims(c)
	if !ok {
		return response.Unauthorized(c, "User authentication required")
	}
	ctx := c.Request().Context()

	guildID, sub, err := h.channelService.Subscribe(ctx, c.Param("slug"), userID)
	if err != nil {
		return handleGuildChannelError(c, "Failed to open event stream", err)
	}
	defer sub.Close()

	res := c.Response()
	rc := http.NewResponseController(res)
	// The server's ReadTimeout would fire mid-stream and cancel the request
	// context; this request has no body left to read, so lift it.
	_ = rc.SetReadDeadline(time.Time{})

	res.Header().Set(echo.HeaderContentType, "text/event-stream")
	res.Header().Set(echo.HeaderCacheControl, "no-cache")
	res.Header().Set("Connection", "keep-alive")
	// Stop nginx-style proxies from buffering the stream.
	res.Header().Set("X-Accel-Buffering", "no")
	res.WriteHeader(http.StatusOK)

	write := func(frame string) error {
		if err := rc.SetWriteDeadline(time.Now().Add(guildStreamWriteTimeout)); err != nil {
			return err
		}
		if _, err := res.Write([]byte(frame)); err != nil {
			return err
		}
		return rc.Flush()
	}

	if err := write(": connected\n\n"); err != nil {
		return nil
	}

	heartbeat := time.NewTicker(guildStreamHeartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case payload, ok := <-sub.C:
			if !ok {
				return nil
			}
			if err := write(fmt.Sprintf("data: %s\n\n", payload)); err != nil {
				return nil
			}
		case <-heartbeat.C:
			member, err := h.channelService.IsMember(ctx, guildID, userID)
			if err != nil {
				guildChatLog.Warn("guild chat: membership re-check failed", "guild_id", guildID, "error", err)
			} else if !member {
				return nil
			}
			if err := write(": ping\n\n"); err != nil {
				return nil
			}
		}
	}
}

func handleGuildChannelError(c *echo.Context, message string, err error) error {
	switch {
	case errors.Is(err, apperrors.ErrGuildNotFound):
		return response.NotFound(c, "Guild not found", err)
	case errors.Is(err, apperrors.ErrGuildNotOwned):
		return response.Forbidden(c, "Access forbidden")
	case errors.Is(err, apperrors.ErrGuildChannelNotFound):
		return response.NotFound(c, "Channel not found", err)
	case errors.Is(err, apperrors.ErrGuildMessageNotFound):
		return response.NotFound(c, "Message not found", err)
	case errors.Is(err, apperrors.ErrNotGuildMember),
		errors.Is(err, apperrors.ErrGuildMessageNotOwned):
		return response.Forbidden(c, "Access forbidden")
	case errors.Is(err, apperrors.ErrGuildChannelNameExists):
		return response.Conflict(c, "Channel name already taken", err.Error())
	case errors.Is(err, apperrors.ErrGuildChannelNameInvalid),
		errors.Is(err, apperrors.ErrGuildMessageEmpty),
		errors.Is(err, apperrors.ErrGuildMessageReplyNotFound),
		errors.Is(err, apperrors.ErrInvalidMessageCursor):
		return response.BadRequest(c, err.Error(), err)
	default:
		return response.InternalServerError(c, message, err)
	}
}
