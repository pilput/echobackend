package service

import (
	"context"
	"errors"
	"testing"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
)

type mockGuildRepo struct {
	createGuildFn        func(ctx context.Context, guild *model.Guild) error
	findGuildByIDFn      func(ctx context.Context, id string) (*model.Guild, error)
	findGuildBySlugFn    func(ctx context.Context, slug string) (*model.Guild, error)
	listPublicGuildsFn   func(ctx context.Context, search string, limit, offset int) ([]*model.Guild, int64, error)
	listGuildsByMemberFn func(ctx context.Context, userID string, limit, offset int) ([]*model.Guild, int64, error)
	updateGuildFn        func(ctx context.Context, id string, updates map[string]any) error
	deleteGuildFn        func(ctx context.Context, id string) error
	addMemberFn          func(ctx context.Context, member *model.GuildMember) error
	findMemberFn         func(ctx context.Context, guildID, userID string) (*model.GuildMember, error)
	removeMemberFn       func(ctx context.Context, guildID, userID string) error
	listMembersFn        func(ctx context.Context, guildID string, limit, offset int) ([]*model.GuildMember, int64, error)
}

func (m *mockGuildRepo) CreateGuild(ctx context.Context, guild *model.Guild) error {
	return m.createGuildFn(ctx, guild)
}

func (m *mockGuildRepo) FindGuildByID(ctx context.Context, id string) (*model.Guild, error) {
	return m.findGuildByIDFn(ctx, id)
}

func (m *mockGuildRepo) FindGuildBySlug(ctx context.Context, slug string) (*model.Guild, error) {
	return m.findGuildBySlugFn(ctx, slug)
}

func (m *mockGuildRepo) ListPublicGuilds(ctx context.Context, search string, limit, offset int) ([]*model.Guild, int64, error) {
	return m.listPublicGuildsFn(ctx, search, limit, offset)
}

func (m *mockGuildRepo) ListGuildsByMember(ctx context.Context, userID string, limit, offset int) ([]*model.Guild, int64, error) {
	return m.listGuildsByMemberFn(ctx, userID, limit, offset)
}

func (m *mockGuildRepo) UpdateGuild(ctx context.Context, id string, updates map[string]any) error {
	return m.updateGuildFn(ctx, id, updates)
}

func (m *mockGuildRepo) DeleteGuild(ctx context.Context, id string) error {
	return m.deleteGuildFn(ctx, id)
}

func (m *mockGuildRepo) AddMember(ctx context.Context, member *model.GuildMember) error {
	return m.addMemberFn(ctx, member)
}

func (m *mockGuildRepo) FindMember(ctx context.Context, guildID, userID string) (*model.GuildMember, error) {
	return m.findMemberFn(ctx, guildID, userID)
}

func (m *mockGuildRepo) RemoveMember(ctx context.Context, guildID, userID string) error {
	return m.removeMemberFn(ctx, guildID, userID)
}

func (m *mockGuildRepo) ListMembers(ctx context.Context, guildID string, limit, offset int) ([]*model.GuildMember, int64, error) {
	return m.listMembersFn(ctx, guildID, limit, offset)
}

// notAMember is the repo behaviour for a user with no membership row.
func notAMember(context.Context, string, string) (*model.GuildMember, error) {
	return nil, apperrors.ErrNotGuildMember
}

func TestCreateGuild_GeneratesSlugFromName(t *testing.T) {
	var created *model.Guild
	repo := &mockGuildRepo{
		createGuildFn: func(_ context.Context, g *model.Guild) error {
			g.ID = "guild-1"
			created = g
			return nil
		},
		findGuildByIDFn: func(context.Context, string) (*model.Guild, error) {
			return created, nil
		},
	}

	svc := NewGuildService(repo)
	resp, err := svc.CreateGuild(context.Background(), "user-1", &dto.CreateGuildRequest{
		Name: "Guild Dokter Indonesia",
	})
	if err != nil {
		t.Fatalf("CreateGuild returned error: %v", err)
	}

	if created.Slug != "guild-dokter-indonesia" {
		t.Errorf("slug = %q, want %q", created.Slug, "guild-dokter-indonesia")
	}
	if !created.IsPublic {
		t.Error("guild should default to public")
	}
	if created.OwnerID != "user-1" {
		t.Errorf("owner = %q, want %q", created.OwnerID, "user-1")
	}
	if resp.MyRole == nil || *resp.MyRole != model.GuildRoleOwner {
		t.Error("creator should be reported as owner")
	}
	if resp.IsMember == nil || !*resp.IsMember {
		t.Error("creator should be reported as a member")
	}
}

func TestCreateGuild_RejectsNameWithoutAlphanumerics(t *testing.T) {
	repo := &mockGuildRepo{
		createGuildFn: func(context.Context, *model.Guild) error {
			t.Fatal("CreateGuild should not reach the repository")
			return nil
		},
	}

	svc := NewGuildService(repo)
	_, err := svc.CreateGuild(context.Background(), "user-1", &dto.CreateGuildRequest{Name: "!!!???"})
	if !errors.Is(err, apperrors.ErrGuildSlugInvalid) {
		t.Errorf("err = %v, want ErrGuildSlugInvalid", err)
	}
}

func TestCreateGuild_RejectsSlugReservedByAStaticRoute(t *testing.T) {
	repo := &mockGuildRepo{
		createGuildFn: func(context.Context, *model.Guild) error {
			t.Fatal("CreateGuild should not reach the repository")
			return nil
		},
	}

	svc := NewGuildService(repo)
	// "Me!" passes the min=3 name validation but slugifies to "me", which
	// GET /api/guilds/me already owns.
	_, err := svc.CreateGuild(context.Background(), "user-1", &dto.CreateGuildRequest{Name: "Me!"})
	if !errors.Is(err, apperrors.ErrGuildSlugReserved) {
		t.Errorf("err = %v, want ErrGuildSlugReserved", err)
	}
}

func TestGetGuildBySlug_PrivateGuildHiddenFromNonMember(t *testing.T) {
	repo := &mockGuildRepo{
		findGuildBySlugFn: func(context.Context, string) (*model.Guild, error) {
			return &model.Guild{ID: "guild-1", Slug: "rahasia", IsPublic: false}, nil
		},
		findMemberFn: notAMember,
	}

	svc := NewGuildService(repo)
	_, err := svc.GetGuildBySlug(context.Background(), "rahasia", "outsider")
	if !errors.Is(err, apperrors.ErrGuildNotFound) {
		t.Errorf("err = %v, want ErrGuildNotFound so the guild does not leak", err)
	}
}

func TestGetGuildBySlug_PrivateGuildVisibleToMember(t *testing.T) {
	repo := &mockGuildRepo{
		findGuildBySlugFn: func(context.Context, string) (*model.Guild, error) {
			return &model.Guild{ID: "guild-1", Slug: "rahasia", IsPublic: false}, nil
		},
		findMemberFn: func(context.Context, string, string) (*model.GuildMember, error) {
			return &model.GuildMember{GuildID: "guild-1", UserID: "member-1", Role: model.GuildRoleMember}, nil
		},
	}

	svc := NewGuildService(repo)
	resp, err := svc.GetGuildBySlug(context.Background(), "rahasia", "member-1")
	if err != nil {
		t.Fatalf("GetGuildBySlug returned error: %v", err)
	}
	if resp.MyRole == nil || *resp.MyRole != model.GuildRoleMember {
		t.Error("member role should be reported")
	}
}

func TestGetGuildBySlug_AnonymousGetsNoMembershipFields(t *testing.T) {
	repo := &mockGuildRepo{
		findGuildBySlugFn: func(context.Context, string) (*model.Guild, error) {
			return &model.Guild{ID: "guild-1", Slug: "publik", IsPublic: true}, nil
		},
		findMemberFn: func(context.Context, string, string) (*model.GuildMember, error) {
			t.Fatal("membership should not be looked up for an anonymous viewer")
			return nil, nil
		},
	}

	svc := NewGuildService(repo)
	resp, err := svc.GetGuildBySlug(context.Background(), "publik", "")
	if err != nil {
		t.Fatalf("GetGuildBySlug returned error: %v", err)
	}
	if resp.IsMember != nil || resp.MyRole != nil {
		t.Error("anonymous viewer should get no membership fields")
	}
}

func TestJoinGuild_RejectsExistingMember(t *testing.T) {
	repo := &mockGuildRepo{
		findGuildBySlugFn: func(context.Context, string) (*model.Guild, error) {
			return &model.Guild{ID: "guild-1", IsPublic: true}, nil
		},
		findMemberFn: func(context.Context, string, string) (*model.GuildMember, error) {
			return &model.GuildMember{Role: model.GuildRoleMember}, nil
		},
		addMemberFn: func(context.Context, *model.GuildMember) error {
			t.Fatal("AddMember should not be called for an existing member")
			return nil
		},
	}

	svc := NewGuildService(repo)
	_, err := svc.JoinGuild(context.Background(), "guild", "user-1")
	if !errors.Is(err, apperrors.ErrAlreadyGuildMember) {
		t.Errorf("err = %v, want ErrAlreadyGuildMember", err)
	}
}

func TestJoinGuild_RejectsPrivateGuild(t *testing.T) {
	repo := &mockGuildRepo{
		findGuildBySlugFn: func(context.Context, string) (*model.Guild, error) {
			return &model.Guild{ID: "guild-1", IsPublic: false}, nil
		},
		addMemberFn: func(context.Context, *model.GuildMember) error {
			t.Fatal("AddMember should not be called for a private guild")
			return nil
		},
	}

	svc := NewGuildService(repo)
	_, err := svc.JoinGuild(context.Background(), "rahasia", "user-1")
	if !errors.Is(err, apperrors.ErrGuildNotFound) {
		t.Errorf("err = %v, want ErrGuildNotFound", err)
	}
}

func TestLeaveGuild_OwnerCannotLeave(t *testing.T) {
	repo := &mockGuildRepo{
		findGuildBySlugFn: func(context.Context, string) (*model.Guild, error) {
			return &model.Guild{ID: "guild-1", OwnerID: "owner-1"}, nil
		},
		findMemberFn: func(context.Context, string, string) (*model.GuildMember, error) {
			return &model.GuildMember{Role: model.GuildRoleOwner}, nil
		},
		removeMemberFn: func(context.Context, string, string) error {
			t.Fatal("RemoveMember should not be called for the owner")
			return nil
		},
	}

	svc := NewGuildService(repo)
	err := svc.LeaveGuild(context.Background(), "guild", "owner-1")
	if !errors.Is(err, apperrors.ErrGuildOwnerCannotLeave) {
		t.Errorf("err = %v, want ErrGuildOwnerCannotLeave", err)
	}
}

func TestUpdateGuild_RejectsPlainMember(t *testing.T) {
	repo := &mockGuildRepo{
		findGuildBySlugFn: func(context.Context, string) (*model.Guild, error) {
			return &model.Guild{ID: "guild-1", OwnerID: "owner-1"}, nil
		},
		findMemberFn: func(context.Context, string, string) (*model.GuildMember, error) {
			return &model.GuildMember{Role: model.GuildRoleMember}, nil
		},
		updateGuildFn: func(context.Context, string, map[string]any) error {
			t.Fatal("UpdateGuild should not be called for a plain member")
			return nil
		},
	}

	name := "Nama Baru"
	svc := NewGuildService(repo)
	_, err := svc.UpdateGuild(context.Background(), "guild", "member-1", &dto.UpdateGuildRequest{Name: &name})
	if !errors.Is(err, apperrors.ErrGuildNotOwned) {
		t.Errorf("err = %v, want ErrGuildNotOwned", err)
	}
}

func TestUpdateGuild_AdminMayUpdateButNotSlug(t *testing.T) {
	var gotUpdates map[string]any
	guild := &model.Guild{ID: "guild-1", Slug: "asli", OwnerID: "owner-1"}
	repo := &mockGuildRepo{
		findGuildBySlugFn: func(context.Context, string) (*model.Guild, error) { return guild, nil },
		findGuildByIDFn:   func(context.Context, string) (*model.Guild, error) { return guild, nil },
		findMemberFn: func(context.Context, string, string) (*model.GuildMember, error) {
			return &model.GuildMember{Role: model.GuildRoleAdmin}, nil
		},
		updateGuildFn: func(_ context.Context, _ string, updates map[string]any) error {
			gotUpdates = updates
			return nil
		},
	}

	name := "Nama Baru"
	svc := NewGuildService(repo)
	if _, err := svc.UpdateGuild(context.Background(), "asli", "admin-1", &dto.UpdateGuildRequest{Name: &name}); err != nil {
		t.Fatalf("UpdateGuild returned error: %v", err)
	}

	if gotUpdates["name"] != "Nama Baru" {
		t.Errorf("name update = %v, want %q", gotUpdates["name"], "Nama Baru")
	}
	if _, ok := gotUpdates["slug"]; ok {
		t.Error("slug must stay immutable")
	}
}

func TestDeleteGuild_OnlyOwner(t *testing.T) {
	repo := &mockGuildRepo{
		findGuildBySlugFn: func(context.Context, string) (*model.Guild, error) {
			return &model.Guild{ID: "guild-1", OwnerID: "owner-1"}, nil
		},
		deleteGuildFn: func(context.Context, string) error {
			t.Fatal("DeleteGuild should not be called by a non-owner")
			return nil
		},
	}

	svc := NewGuildService(repo)
	if err := svc.DeleteGuild(context.Background(), "guild", "admin-1"); !errors.Is(err, apperrors.ErrGuildNotOwned) {
		t.Errorf("err = %v, want ErrGuildNotOwned", err)
	}
}
