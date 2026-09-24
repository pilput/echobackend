package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
	"echobackend/internal/platform/realtime"
	"echobackend/internal/repository"
)

const (
	testGuildID   = "00000000-0000-7000-8000-000000000001"
	testChannelID = "00000000-0000-7000-8000-000000000002"
	testMessageID = "00000000-0000-7000-8000-000000000003"
)

type mockGuildChannelRepo struct {
	resolveAccessFn        func(ctx context.Context, guildSlug, userID, channelID string) (*repository.GuildChannelAccess, error)
	isGuildMemberFn        func(ctx context.Context, guildID, userID string) (bool, error)
	createChannelFn        func(ctx context.Context, channel *model.GuildChannel) error
	findChannelFn          func(ctx context.Context, guildID, channelID string) (*model.GuildChannel, error)
	listChannelsFn         func(ctx context.Context, guildID string) ([]*model.GuildChannel, error)
	updateChannelFn        func(ctx context.Context, guildID, channelID string, updates map[string]any) error
	deleteChannelFn        func(ctx context.Context, guildID, channelID string) error
	createMessageFn        func(ctx context.Context, message *model.GuildChannelMessage) error
	findMessageFn          func(ctx context.Context, channelID, messageID string) (*model.GuildChannelMessage, error)
	findMessageAuthorFn    func(ctx context.Context, channelID, messageID string) (string, error)
	messageExistsFn        func(ctx context.Context, channelID, messageID string) (bool, error)
	listMessagesFn         func(ctx context.Context, channelID, before string, limit int) ([]*model.GuildChannelMessage, error)
	updateMessageContentFn func(ctx context.Context, channelID, messageID, content string, editedAt time.Time) error
	deleteMessageFn        func(ctx context.Context, channelID, messageID string) error
}

func (m *mockGuildChannelRepo) ResolveAccess(ctx context.Context, guildSlug, userID, channelID string) (*repository.GuildChannelAccess, error) {
	return m.resolveAccessFn(ctx, guildSlug, userID, channelID)
}

func (m *mockGuildChannelRepo) IsGuildMember(ctx context.Context, guildID, userID string) (bool, error) {
	return m.isGuildMemberFn(ctx, guildID, userID)
}

func (m *mockGuildChannelRepo) CreateChannel(ctx context.Context, channel *model.GuildChannel) error {
	return m.createChannelFn(ctx, channel)
}

func (m *mockGuildChannelRepo) FindChannel(ctx context.Context, guildID, channelID string) (*model.GuildChannel, error) {
	return m.findChannelFn(ctx, guildID, channelID)
}

func (m *mockGuildChannelRepo) ListChannels(ctx context.Context, guildID string) ([]*model.GuildChannel, error) {
	return m.listChannelsFn(ctx, guildID)
}

func (m *mockGuildChannelRepo) UpdateChannel(ctx context.Context, guildID, channelID string, updates map[string]any) error {
	return m.updateChannelFn(ctx, guildID, channelID, updates)
}

func (m *mockGuildChannelRepo) DeleteChannel(ctx context.Context, guildID, channelID string) error {
	return m.deleteChannelFn(ctx, guildID, channelID)
}

func (m *mockGuildChannelRepo) CreateMessage(ctx context.Context, message *model.GuildChannelMessage) error {
	return m.createMessageFn(ctx, message)
}

func (m *mockGuildChannelRepo) FindMessage(ctx context.Context, channelID, messageID string) (*model.GuildChannelMessage, error) {
	return m.findMessageFn(ctx, channelID, messageID)
}

func (m *mockGuildChannelRepo) FindMessageAuthor(ctx context.Context, channelID, messageID string) (string, error) {
	return m.findMessageAuthorFn(ctx, channelID, messageID)
}

func (m *mockGuildChannelRepo) MessageExists(ctx context.Context, channelID, messageID string) (bool, error) {
	return m.messageExistsFn(ctx, channelID, messageID)
}

func (m *mockGuildChannelRepo) ListMessages(ctx context.Context, channelID, before string, limit int) ([]*model.GuildChannelMessage, error) {
	return m.listMessagesFn(ctx, channelID, before, limit)
}

func (m *mockGuildChannelRepo) UpdateMessageContent(ctx context.Context, channelID, messageID, content string, editedAt time.Time) error {
	return m.updateMessageContentFn(ctx, channelID, messageID, content, editedAt)
}

func (m *mockGuildChannelRepo) DeleteMessage(ctx context.Context, channelID, messageID string) error {
	return m.deleteMessageFn(ctx, channelID, messageID)
}

type publishedEvent struct {
	topic string
	event dto.GuildEvent
}

type recordingBroker struct {
	events []publishedEvent
}

func (b *recordingBroker) Publish(_ context.Context, topic string, event any) error {
	b.events = append(b.events, publishedEvent{topic: topic, event: event.(dto.GuildEvent)})
	return nil
}

func (b *recordingBroker) Subscribe(string) *realtime.Subscription {
	return nil
}

// withAccess returns a repo whose ResolveAccess reports a guild in which the
// caller holds role ("" = not a member) and in which testChannelID exists.
func withAccess(isPublic bool, role string) *mockGuildChannelRepo {
	return &mockGuildChannelRepo{
		resolveAccessFn: func(_ context.Context, _, _, channelID string) (*repository.GuildChannelAccess, error) {
			access := &repository.GuildChannelAccess{GuildID: testGuildID, IsPublic: isPublic, Role: role}
			if channelID == testChannelID {
				ch := channelID
				access.ChannelID = &ch
			}
			return access, nil
		},
	}
}

func TestSendMessage_RequiresMembership(t *testing.T) {
	repo := withAccess(true, "")
	repo.createMessageFn = func(context.Context, *model.GuildChannelMessage) error {
		t.Fatal("non-member must not be able to post")
		return nil
	}
	svc := NewGuildChannelService(repo, &recordingBroker{})

	_, err := svc.SendMessage(context.Background(), "g", testChannelID, "user-1", &dto.CreateGuildMessageRequest{Content: "hi"})
	if !errors.Is(err, apperrors.ErrNotGuildMember) {
		t.Fatalf("err = %v, want ErrNotGuildMember", err)
	}
}

func TestSendMessage_PrivateGuildHiddenFromNonMembers(t *testing.T) {
	svc := NewGuildChannelService(withAccess(false, ""), &recordingBroker{})

	// Even a malformed channel id must not reveal that the guild exists.
	for _, channelID := range []string{testChannelID, "not-a-uuid"} {
		_, err := svc.SendMessage(context.Background(), "g", channelID, "user-1", &dto.CreateGuildMessageRequest{Content: "hi"})
		if !errors.Is(err, apperrors.ErrGuildNotFound) {
			t.Fatalf("channel %q: err = %v, want ErrGuildNotFound", channelID, err)
		}
	}
}

func TestSendMessage_UnknownGuild(t *testing.T) {
	repo := &mockGuildChannelRepo{
		resolveAccessFn: func(context.Context, string, string, string) (*repository.GuildChannelAccess, error) {
			return nil, apperrors.ErrGuildNotFound
		},
	}
	svc := NewGuildChannelService(repo, &recordingBroker{})

	_, err := svc.SendMessage(context.Background(), "nope", testChannelID, "user-1", &dto.CreateGuildMessageRequest{Content: "hi"})
	if !errors.Is(err, apperrors.ErrGuildNotFound) {
		t.Fatalf("err = %v, want ErrGuildNotFound", err)
	}
}

func TestSendMessage_TrimsContentAndPublishes(t *testing.T) {
	var stored *model.GuildChannelMessage
	repo := withAccess(true, model.GuildRoleMember)
	repo.createMessageFn = func(_ context.Context, m *model.GuildChannelMessage) error {
		m.ID = testMessageID
		stored = m
		return nil
	}
	repo.findMessageFn = func(context.Context, string, string) (*model.GuildChannelMessage, error) {
		return stored, nil
	}
	broker := &recordingBroker{}
	svc := NewGuildChannelService(repo, broker)

	resp, err := svc.SendMessage(context.Background(), "g", testChannelID, "user-1", &dto.CreateGuildMessageRequest{Content: "  halo semua  "})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if stored.Content != "halo semua" {
		t.Errorf("stored content = %q", stored.Content)
	}
	if resp.GuildID != testGuildID || resp.ChannelID != testChannelID || resp.AuthorID != "user-1" {
		t.Errorf("unexpected response: %+v", resp)
	}

	if len(broker.events) != 1 {
		t.Fatalf("published %d events, want 1", len(broker.events))
	}
	got := broker.events[0]
	if got.topic != GuildEventTopic(testGuildID) || got.event.Type != dto.GuildEventMessageCreated || got.event.ChannelID != testChannelID {
		t.Errorf("unexpected event: %+v", got)
	}
}

func TestSendMessage_RejectsBlankContent(t *testing.T) {
	svc := NewGuildChannelService(withAccess(true, model.GuildRoleMember), &recordingBroker{})

	_, err := svc.SendMessage(context.Background(), "g", testChannelID, "user-1", &dto.CreateGuildMessageRequest{Content: " \n\t "})
	if !errors.Is(err, apperrors.ErrGuildMessageEmpty) {
		t.Fatalf("err = %v, want ErrGuildMessageEmpty", err)
	}
}

func TestSendMessage_ReplyMustBeInSameChannel(t *testing.T) {
	repo := withAccess(true, model.GuildRoleMember)
	repo.messageExistsFn = func(context.Context, string, string) (bool, error) { return false, nil }
	repo.createMessageFn = func(context.Context, *model.GuildChannelMessage) error {
		t.Fatal("a reply to an unknown message must not be stored")
		return nil
	}
	svc := NewGuildChannelService(repo, &recordingBroker{})

	parent := testMessageID
	_, err := svc.SendMessage(context.Background(), "g", testChannelID, "user-1", &dto.CreateGuildMessageRequest{Content: "hi", ReplyToID: &parent})
	if !errors.Is(err, apperrors.ErrGuildMessageReplyNotFound) {
		t.Fatalf("err = %v, want ErrGuildMessageReplyNotFound", err)
	}
}

func TestSendMessage_UnknownChannelIsNotFound(t *testing.T) {
	svc := NewGuildChannelService(withAccess(true, model.GuildRoleMember), &recordingBroker{})

	for _, channelID := range []string{"not-a-uuid", "00000000-0000-7000-8000-00000000ffff"} {
		_, err := svc.SendMessage(context.Background(), "g", channelID, "user-1", &dto.CreateGuildMessageRequest{Content: "hi"})
		if !errors.Is(err, apperrors.ErrGuildChannelNotFound) {
			t.Fatalf("channel %q: err = %v, want ErrGuildChannelNotFound", channelID, err)
		}
	}
}

func TestEditMessage_OnlyAuthorEvenForAdmins(t *testing.T) {
	repo := withAccess(true, model.GuildRoleAdmin)
	repo.findMessageAuthorFn = func(context.Context, string, string) (string, error) { return "someone-else", nil }
	repo.updateMessageContentFn = func(context.Context, string, string, string, time.Time) error {
		t.Fatal("an admin must not edit someone else's message")
		return nil
	}
	svc := NewGuildChannelService(repo, &recordingBroker{})

	_, err := svc.EditMessage(context.Background(), "g", testChannelID, testMessageID, "admin-1", &dto.UpdateGuildMessageRequest{Content: "x"})
	if !errors.Is(err, apperrors.ErrGuildMessageNotOwned) {
		t.Fatalf("err = %v, want ErrGuildMessageNotOwned", err)
	}
}

func TestDeleteMessage_Permissions(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		wantErr error
	}{
		{name: "admin may delete others' messages", role: model.GuildRoleAdmin},
		{name: "owner may delete others' messages", role: model.GuildRoleOwner},
		{name: "member may not delete others' messages", role: model.GuildRoleMember, wantErr: apperrors.ErrGuildMessageNotOwned},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleted := false
			repo := withAccess(true, tt.role)
			repo.findMessageAuthorFn = func(context.Context, string, string) (string, error) { return "author", nil }
			repo.deleteMessageFn = func(context.Context, string, string) error {
				deleted = true
				return nil
			}
			broker := &recordingBroker{}
			svc := NewGuildChannelService(repo, broker)

			err := svc.DeleteMessage(context.Background(), "g", testChannelID, testMessageID, "user-1")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if deleted != (tt.wantErr == nil) {
				t.Errorf("deleted = %v", deleted)
			}
			if tt.wantErr == nil && (len(broker.events) != 1 || broker.events[0].event.Type != dto.GuildEventMessageDeleted) {
				t.Errorf("expected one message.deleted event, got %+v", broker.events)
			}
		})
	}
}

func TestDeleteMessage_MalformedIDIsNotFound(t *testing.T) {
	repo := withAccess(true, model.GuildRoleOwner)
	repo.findMessageAuthorFn = func(context.Context, string, string) (string, error) {
		t.Fatal("a malformed id must not reach the database")
		return "", nil
	}
	svc := NewGuildChannelService(repo, &recordingBroker{})

	err := svc.DeleteMessage(context.Background(), "g", testChannelID, "bad", "user-1")
	if !errors.Is(err, apperrors.ErrGuildMessageNotFound) {
		t.Fatalf("err = %v, want ErrGuildMessageNotFound", err)
	}
}

func TestCreateChannel_RequiresManager(t *testing.T) {
	svc := NewGuildChannelService(withAccess(true, model.GuildRoleMember), &recordingBroker{})

	_, err := svc.CreateChannel(context.Background(), "g", "user-1", &dto.CreateGuildChannelRequest{Name: "random"})
	if !errors.Is(err, apperrors.ErrGuildNotOwned) {
		t.Fatalf("err = %v, want ErrGuildNotOwned", err)
	}
}

func TestCreateChannel_NormalisesName(t *testing.T) {
	repo := withAccess(true, model.GuildRoleAdmin)
	repo.createChannelFn = func(_ context.Context, ch *model.GuildChannel) error {
		ch.ID = testChannelID
		return nil
	}
	broker := &recordingBroker{}
	svc := NewGuildChannelService(repo, broker)

	resp, err := svc.CreateChannel(context.Background(), "g", "admin-1", &dto.CreateGuildChannelRequest{Name: "  Diskusi Umum! "})
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}
	if resp.Name != "diskusi-umum" || resp.ID != testChannelID || resp.GuildID != testGuildID {
		t.Errorf("unexpected response: %+v", resp)
	}
	if len(broker.events) != 1 || broker.events[0].event.Type != dto.GuildEventChannelCreated {
		t.Errorf("expected one channel.created event, got %+v", broker.events)
	}
}

func TestCreateChannel_RejectsNameWithoutAlphanumerics(t *testing.T) {
	svc := NewGuildChannelService(withAccess(true, model.GuildRoleOwner), &recordingBroker{})

	_, err := svc.CreateChannel(context.Background(), "g", "admin-1", &dto.CreateGuildChannelRequest{Name: "!!!"})
	if !errors.Is(err, apperrors.ErrGuildChannelNameInvalid) {
		t.Fatalf("err = %v, want ErrGuildChannelNameInvalid", err)
	}
}

func TestDeleteChannel_UnknownChannelIsNotFound(t *testing.T) {
	repo := withAccess(true, model.GuildRoleOwner)
	repo.deleteChannelFn = func(context.Context, string, string) error {
		t.Fatal("unknown channel must not reach DeleteChannel")
		return nil
	}
	svc := NewGuildChannelService(repo, &recordingBroker{})

	err := svc.DeleteChannel(context.Background(), "g", "00000000-0000-7000-8000-00000000ffff", "owner-1")
	if !errors.Is(err, apperrors.ErrGuildChannelNotFound) {
		t.Fatalf("err = %v, want ErrGuildChannelNotFound", err)
	}
}

func TestListChannels_PrivateGuildHiddenFromNonMembers(t *testing.T) {
	svc := NewGuildChannelService(withAccess(false, ""), &recordingBroker{})

	_, err := svc.ListChannels(context.Background(), "g", "")
	if !errors.Is(err, apperrors.ErrGuildNotFound) {
		t.Fatalf("err = %v, want ErrGuildNotFound", err)
	}
}

func TestListMessages_CursorMeta(t *testing.T) {
	ids := []string{
		"00000000-0000-7000-8000-000000000013",
		"00000000-0000-7000-8000-000000000012",
		"00000000-0000-7000-8000-000000000011",
	}
	var gotLimit int
	repo := withAccess(true, model.GuildRoleMember)
	repo.listMessagesFn = func(_ context.Context, _ string, _ string, limit int) ([]*model.GuildChannelMessage, error) {
		gotLimit = limit
		out := make([]*model.GuildChannelMessage, 0, len(ids))
		for _, id := range ids[:min(limit, len(ids))] {
			out = append(out, &model.GuildChannelMessage{ID: id, ChannelID: testChannelID})
		}
		return out, nil
	}
	svc := NewGuildChannelService(repo, &recordingBroker{})

	msgs, meta, err := svc.ListMessages(context.Background(), "g", testChannelID, "user-1", "", 2)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if gotLimit != 3 {
		t.Errorf("repo limit = %d, want limit+1 = 3", gotLimit)
	}
	if len(msgs) != 2 || !meta.HasMore || meta.NextBefore == nil || *meta.NextBefore != ids[1] {
		t.Fatalf("unexpected page: %d messages, meta %+v", len(msgs), meta)
	}

	msgs, meta, err = svc.ListMessages(context.Background(), "g", testChannelID, "user-1", "", 5)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(msgs) != 3 || meta.HasMore || meta.NextBefore != nil {
		t.Fatalf("last page: %d messages, meta %+v", len(msgs), meta)
	}
}

func TestListMessages_RejectsInvalidCursor(t *testing.T) {
	svc := NewGuildChannelService(&mockGuildChannelRepo{}, &recordingBroker{})

	_, _, err := svc.ListMessages(context.Background(), "g", testChannelID, "user-1", "nope", 10)
	if !errors.Is(err, apperrors.ErrInvalidMessageCursor) {
		t.Fatalf("err = %v, want ErrInvalidMessageCursor", err)
	}
}

func TestGuildMessageToResponse_TruncatesReplyQuote(t *testing.T) {
	parent := &model.GuildChannelMessage{ID: testMessageID, AuthorID: "a", Content: strings.Repeat("é", dto.GuildMessageReplyPreviewLen+50)}
	resp := dto.GuildMessageToResponse(&model.GuildChannelMessage{ID: "m", ReplyTo: parent}, testGuildID)

	if got := []rune(resp.ReplyTo.Content); len(got) != dto.GuildMessageReplyPreviewLen {
		t.Fatalf("quote has %d runes, want %d", len(got), dto.GuildMessageReplyPreviewLen)
	}
}

func TestSubscribe_ReceivesPublishedMessages(t *testing.T) {
	hub := realtime.NewHub(nil)
	hub.Start()
	defer func() { _ = hub.Close() }()

	var stored *model.GuildChannelMessage
	repo := withAccess(true, model.GuildRoleMember)
	repo.createMessageFn = func(_ context.Context, m *model.GuildChannelMessage) error {
		m.ID = testMessageID
		stored = m
		return nil
	}
	repo.findMessageFn = func(context.Context, string, string) (*model.GuildChannelMessage, error) {
		return stored, nil
	}
	svc := NewGuildChannelService(repo, hub)

	guildID, sub, err := svc.Subscribe(context.Background(), "g", "user-1")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Close()
	if guildID != testGuildID {
		t.Errorf("guildID = %q", guildID)
	}

	if _, err := svc.SendMessage(context.Background(), "g", testChannelID, "user-1", &dto.CreateGuildMessageRequest{Content: "hi"}); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	select {
	case payload := <-sub.C:
		var event struct {
			Type string                   `json:"type"`
			Data dto.GuildMessageResponse `json:"data"`
		}
		if err := json.Unmarshal(payload, &event); err != nil {
			t.Fatalf("decode event: %v", err)
		}
		if event.Type != dto.GuildEventMessageCreated || event.Data.Content != "hi" {
			t.Errorf("unexpected event: %s", payload)
		}
	default:
		t.Fatal("subscriber received nothing")
	}
}

func TestSubscribe_RequiresMembership(t *testing.T) {
	svc := NewGuildChannelService(withAccess(true, ""), &recordingBroker{})

	if _, _, err := svc.Subscribe(context.Background(), "g", "user-1"); !errors.Is(err, apperrors.ErrNotGuildMember) {
		t.Fatalf("err = %v, want ErrNotGuildMember", err)
	}
}
