package service

import (
	"context"
	"strings"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
	"echobackend/internal/platform/realtime"
	"echobackend/internal/repository"
	"echobackend/pkg/slug"
	"echobackend/pkg/validator"
)

// guildChannelNameMaxLen mirrors the varchar(100) column on guild_channels.name.
const guildChannelNameMaxLen = 100

// GuildEventBroker delivers realtime guild events to SSE subscribers.
// *realtime.Hub implements it.
type GuildEventBroker interface {
	Publish(ctx context.Context, topic string, event any) error
	Subscribe(topic string) *realtime.Subscription
}

// GuildChannelService runs the Discord-style text channels of a guild.
//
// Access rules:
//   - the channel list is visible to whoever can see the guild;
//   - reading, sending and streaming messages require membership;
//   - creating, renaming and deleting channels requires owner or admin;
//   - a message may be edited by its author only, and deleted by its author or
//     a guild owner/admin.
//
// A private guild is reported as not found to non-members, as in GuildService.
type GuildChannelService interface {
	ListChannels(ctx context.Context, guildSlug, viewerID string) ([]*dto.GuildChannelResponse, error)
	CreateChannel(ctx context.Context, guildSlug, userID string, req *dto.CreateGuildChannelRequest) (*dto.GuildChannelResponse, error)
	UpdateChannel(ctx context.Context, guildSlug, channelID, userID string, req *dto.UpdateGuildChannelRequest) (*dto.GuildChannelResponse, error)
	DeleteChannel(ctx context.Context, guildSlug, channelID, userID string) error

	ListMessages(ctx context.Context, guildSlug, channelID, userID, before string, limit int) ([]*dto.GuildMessageResponse, *dto.GuildMessageCursorMeta, error)
	SendMessage(ctx context.Context, guildSlug, channelID, userID string, req *dto.CreateGuildMessageRequest) (*dto.GuildMessageResponse, error)
	EditMessage(ctx context.Context, guildSlug, channelID, messageID, userID string, req *dto.UpdateGuildMessageRequest) (*dto.GuildMessageResponse, error)
	DeleteMessage(ctx context.Context, guildSlug, channelID, messageID, userID string) error

	// Subscribe opens the realtime event stream of a guild for a member. The
	// returned guild id lets the caller re-check membership with IsMember while
	// the stream stays open.
	Subscribe(ctx context.Context, guildSlug, userID string) (string, *realtime.Subscription, error)
	IsMember(ctx context.Context, guildID, userID string) (bool, error)
}

// Every request starts with a single ResolveAccess query that yields the
// guild, the caller's role and the channel together, instead of separate
// guild, member and channel lookups.
type guildChannelService struct {
	channelRepo repository.GuildChannelRepository
	broker      GuildEventBroker
}

func NewGuildChannelService(channelRepo repository.GuildChannelRepository, broker GuildEventBroker) GuildChannelService {
	return &guildChannelService{channelRepo: channelRepo, broker: broker}
}

// GuildEventTopic is the realtime topic carrying every event of one guild.
func GuildEventTopic(guildID string) string {
	return "guild:" + guildID
}

func (s *guildChannelService) ListChannels(ctx context.Context, guildSlug, viewerID string) ([]*dto.GuildChannelResponse, error) {
	access, err := s.resolveGuild(ctx, guildSlug, viewerID, "")
	if err != nil {
		return nil, err
	}
	channels, err := s.channelRepo.ListChannels(ctx, access.GuildID)
	if err != nil {
		return nil, err
	}
	out := make([]*dto.GuildChannelResponse, 0, len(channels))
	for _, ch := range channels {
		out = append(out, dto.GuildChannelToResponse(ch))
	}
	return out, nil
}

func (s *guildChannelService) CreateChannel(ctx context.Context, guildSlug, userID string, req *dto.CreateGuildChannelRequest) (*dto.GuildChannelResponse, error) {
	access, err := s.requireManager(ctx, guildSlug, userID, "")
	if err != nil {
		return nil, err
	}
	name := slug.Make(req.Name, guildChannelNameMaxLen)
	if name == "" {
		return nil, apperrors.ErrGuildChannelNameInvalid
	}

	channel := &model.GuildChannel{
		GuildID:   access.GuildID,
		Name:      name,
		Topic:     req.Topic,
		CreatedBy: &userID,
	}
	if req.Position != nil {
		channel.Position = *req.Position
	}
	// Create reads id and timestamps back via RETURNING; no re-fetch needed.
	if err := s.channelRepo.CreateChannel(ctx, channel); err != nil {
		return nil, err
	}

	resp := dto.GuildChannelToResponse(channel)
	s.publish(ctx, access.GuildID, channel.ID, dto.GuildEventChannelCreated, resp)
	return resp, nil
}

func (s *guildChannelService) UpdateChannel(ctx context.Context, guildSlug, channelID, userID string, req *dto.UpdateGuildChannelRequest) (*dto.GuildChannelResponse, error) {
	if !validator.IsValidUUID(channelID) {
		channelID = ""
	}
	access, err := s.requireManager(ctx, guildSlug, userID, channelID)
	if err != nil {
		return nil, err
	}
	if access.ChannelID == nil {
		return nil, apperrors.ErrGuildChannelNotFound
	}

	updates := make(map[string]any)
	if req.Name != nil {
		name := slug.Make(*req.Name, guildChannelNameMaxLen)
		if name == "" {
			return nil, apperrors.ErrGuildChannelNameInvalid
		}
		updates["name"] = name
	}
	if req.Topic != nil {
		updates["topic"] = *req.Topic
	}
	if req.Position != nil {
		updates["position"] = *req.Position
	}
	if err := s.channelRepo.UpdateChannel(ctx, access.GuildID, channelID, updates); err != nil {
		return nil, err
	}

	updated, err := s.channelRepo.FindChannel(ctx, access.GuildID, channelID)
	if err != nil {
		return nil, err
	}
	resp := dto.GuildChannelToResponse(updated)
	if len(updates) > 0 {
		s.publish(ctx, access.GuildID, channelID, dto.GuildEventChannelUpdated, resp)
	}
	return resp, nil
}

func (s *guildChannelService) DeleteChannel(ctx context.Context, guildSlug, channelID, userID string) error {
	if !validator.IsValidUUID(channelID) {
		channelID = ""
	}
	access, err := s.requireManager(ctx, guildSlug, userID, channelID)
	if err != nil {
		return err
	}
	if access.ChannelID == nil {
		return apperrors.ErrGuildChannelNotFound
	}
	if err := s.channelRepo.DeleteChannel(ctx, access.GuildID, channelID); err != nil {
		return err
	}
	s.publish(ctx, access.GuildID, channelID, dto.GuildEventChannelDeleted, map[string]string{"id": channelID})
	return nil
}

func (s *guildChannelService) ListMessages(ctx context.Context, guildSlug, channelID, userID, before string, limit int) ([]*dto.GuildMessageResponse, *dto.GuildMessageCursorMeta, error) {
	if before != "" && !validator.IsValidUUID(before) {
		return nil, nil, apperrors.ErrInvalidMessageCursor
	}
	access, err := s.resolveMemberChannel(ctx, guildSlug, channelID, userID)
	if err != nil {
		return nil, nil, err
	}

	// One extra row tells whether an older page exists without a COUNT.
	messages, err := s.channelRepo.ListMessages(ctx, channelID, before, limit+1)
	if err != nil {
		return nil, nil, err
	}
	meta := &dto.GuildMessageCursorMeta{Limit: limit}
	if len(messages) > limit {
		messages = messages[:limit]
		meta.HasMore = true
		next := messages[len(messages)-1].ID
		meta.NextBefore = &next
	}

	out := make([]*dto.GuildMessageResponse, 0, len(messages))
	for _, m := range messages {
		out = append(out, dto.GuildMessageToResponse(m, access.GuildID))
	}
	return out, meta, nil
}

func (s *guildChannelService) SendMessage(ctx context.Context, guildSlug, channelID, userID string, req *dto.CreateGuildMessageRequest) (*dto.GuildMessageResponse, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, apperrors.ErrGuildMessageEmpty
	}
	access, err := s.resolveMemberChannel(ctx, guildSlug, channelID, userID)
	if err != nil {
		return nil, err
	}

	message := &model.GuildChannelMessage{
		ChannelID: channelID,
		AuthorID:  userID,
		Content:   content,
	}
	if req.ReplyToID != nil && *req.ReplyToID != "" {
		exists, err := s.channelRepo.MessageExists(ctx, channelID, *req.ReplyToID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, apperrors.ErrGuildMessageReplyNotFound
		}
		message.ReplyToID = req.ReplyToID
	}
	if err := s.channelRepo.CreateMessage(ctx, message); err != nil {
		return nil, err
	}

	// Re-read for the author brief and reply quote that the response and the
	// event carry.
	created, err := s.channelRepo.FindMessage(ctx, channelID, message.ID)
	if err != nil {
		return nil, err
	}
	resp := dto.GuildMessageToResponse(created, access.GuildID)
	s.publish(ctx, access.GuildID, channelID, dto.GuildEventMessageCreated, resp)
	return resp, nil
}

func (s *guildChannelService) EditMessage(ctx context.Context, guildSlug, channelID, messageID, userID string, req *dto.UpdateGuildMessageRequest) (*dto.GuildMessageResponse, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, apperrors.ErrGuildMessageEmpty
	}
	access, err := s.resolveMemberChannel(ctx, guildSlug, channelID, userID)
	if err != nil {
		return nil, err
	}
	authorID, err := s.findMessageAuthor(ctx, channelID, messageID)
	if err != nil {
		return nil, err
	}
	// Moderators may delete other people's messages but never put words in
	// their mouth.
	if authorID != userID {
		return nil, apperrors.ErrGuildMessageNotOwned
	}

	if err := s.channelRepo.UpdateMessageContent(ctx, channelID, messageID, content, time.Now()); err != nil {
		return nil, err
	}
	updated, err := s.channelRepo.FindMessage(ctx, channelID, messageID)
	if err != nil {
		return nil, err
	}
	resp := dto.GuildMessageToResponse(updated, access.GuildID)
	s.publish(ctx, access.GuildID, channelID, dto.GuildEventMessageUpdated, resp)
	return resp, nil
}

func (s *guildChannelService) DeleteMessage(ctx context.Context, guildSlug, channelID, messageID, userID string) error {
	access, err := s.resolveMemberChannel(ctx, guildSlug, channelID, userID)
	if err != nil {
		return err
	}
	authorID, err := s.findMessageAuthor(ctx, channelID, messageID)
	if err != nil {
		return err
	}
	if authorID != userID && !isGuildManager(access.Role) {
		return apperrors.ErrGuildMessageNotOwned
	}

	if err := s.channelRepo.DeleteMessage(ctx, channelID, messageID); err != nil {
		return err
	}
	s.publish(ctx, access.GuildID, channelID, dto.GuildEventMessageDeleted, map[string]string{
		"id":         messageID,
		"channel_id": channelID,
	})
	return nil
}

func (s *guildChannelService) Subscribe(ctx context.Context, guildSlug, userID string) (string, *realtime.Subscription, error) {
	access, err := s.resolveGuild(ctx, guildSlug, userID, "")
	if err != nil {
		return "", nil, err
	}
	if access.Role == "" {
		return "", nil, apperrors.ErrNotGuildMember
	}
	return access.GuildID, s.broker.Subscribe(GuildEventTopic(access.GuildID)), nil
}

func (s *guildChannelService) IsMember(ctx context.Context, guildID, userID string) (bool, error) {
	return s.channelRepo.IsGuildMember(ctx, guildID, userID)
}

// resolveGuild reads the caller's access in one query and hides private
// guilds from non-members. channelID may be empty.
func (s *guildChannelService) resolveGuild(ctx context.Context, guildSlug, userID, channelID string) (*repository.GuildChannelAccess, error) {
	access, err := s.channelRepo.ResolveAccess(ctx, guildSlug, userID, channelID)
	if err != nil {
		return nil, err
	}
	if !access.IsPublic && access.Role == "" {
		return nil, apperrors.ErrGuildNotFound
	}
	return access, nil
}

// requireManager checks for an owner or admin. The caller checks
// access.ChannelID itself when it passed a channel.
func (s *guildChannelService) requireManager(ctx context.Context, guildSlug, userID, channelID string) (*repository.GuildChannelAccess, error) {
	access, err := s.resolveGuild(ctx, guildSlug, userID, channelID)
	if err != nil {
		return nil, err
	}
	if !isGuildManager(access.Role) {
		return nil, apperrors.ErrGuildNotOwned
	}
	return access, nil
}

// resolveMemberChannel checks that the caller is a member of the guild and
// that the channel belongs to it.
func (s *guildChannelService) resolveMemberChannel(ctx context.Context, guildSlug, channelID, userID string) (*repository.GuildChannelAccess, error) {
	// A malformed id is looked up as "no channel", so the guild checks still
	// run first and a hidden guild keeps reading as missing.
	if !validator.IsValidUUID(channelID) {
		channelID = ""
	}
	access, err := s.resolveGuild(ctx, guildSlug, userID, channelID)
	if err != nil {
		return nil, err
	}
	if access.Role == "" {
		return nil, apperrors.ErrNotGuildMember
	}
	if access.ChannelID == nil {
		return nil, apperrors.ErrGuildChannelNotFound
	}
	return access, nil
}

func (s *guildChannelService) findMessageAuthor(ctx context.Context, channelID, messageID string) (string, error) {
	if !validator.IsValidUUID(messageID) {
		return "", apperrors.ErrGuildMessageNotFound
	}
	return s.channelRepo.FindMessageAuthor(ctx, channelID, messageID)
}

// publish pushes an event to live subscribers. The change is already
// committed, so a delivery failure is logged rather than returned: clients
// recover by refetching history when they reconnect.
func (s *guildChannelService) publish(ctx context.Context, guildID, channelID, eventType string, data any) {
	if s.broker == nil {
		return
	}
	event := dto.GuildEvent{Type: eventType, GuildID: guildID, ChannelID: channelID, Data: data}
	if err := s.broker.Publish(ctx, GuildEventTopic(guildID), event); err != nil {
		guildChatLog.Warn("guild chat: failed to publish event", "type", eventType, "guild_id", guildID, "error", err)
	}
}

func isGuildManager(role string) bool {
	return role == model.GuildRoleOwner || role == model.GuildRoleAdmin
}
