package guilds

import (
	"context"
	"errors"

	"echobackend/pkg/slug"
)

// guildSlugMaxLen mirrors the varchar(100) column on guilds.slug.
const guildSlugMaxLen = 100

// reservedGuildSlugs are taken by static routes under /api/guilds. A guild
// holding one of these would be created but unreachable by URL, so creation is
// rejected instead. Keep in sync with internal/routes/guild_routes.go.
var reservedGuildSlugs = map[string]struct{}{
	"me": {},
}

type GuildService interface {
	CreateGuild(ctx context.Context, ownerID string, req *CreateGuildRequest) (*GuildResponse, error)
	GetGuildBySlug(ctx context.Context, guildSlug, viewerID string) (*GuildResponse, error)
	ListGuilds(ctx context.Context, search string, limit, offset int) ([]*GuildResponse, int64, error)
	ListMyGuilds(ctx context.Context, userID string, limit, offset int) ([]*GuildResponse, int64, error)
	UpdateGuild(ctx context.Context, guildSlug, userID string, req *UpdateGuildRequest) (*GuildResponse, error)
	DeleteGuild(ctx context.Context, guildSlug, userID string) error

	JoinGuild(ctx context.Context, guildSlug, userID string) (*GuildMemberResponse, error)
	LeaveGuild(ctx context.Context, guildSlug, userID string) error
	ListMembers(ctx context.Context, guildSlug, viewerID string, limit, offset int) ([]*GuildMemberResponse, int64, error)
}

type guildService struct {
	guildRepo GuildRepository
}

func NewGuildService(guildRepo GuildRepository) GuildService {
	return &guildService{guildRepo: guildRepo}
}

func (s *guildService) CreateGuild(ctx context.Context, ownerID string, req *CreateGuildRequest) (*GuildResponse, error) {
	if err := rejectSpamText(req.Name, optionalText(req.Slug), optionalText(req.Description)); err != nil {
		return nil, err
	}

	desired := req.Name
	if req.Slug != nil && *req.Slug != "" {
		desired = *req.Slug
	}
	generated := slug.Make(desired, guildSlugMaxLen)
	if generated == "" {
		return nil, ErrGuildSlugInvalid
	}
	if _, reserved := reservedGuildSlugs[generated]; reserved {
		return nil, ErrGuildSlugReserved
	}

	isPublic := true
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}

	guild := &Guild{
		OwnerID:     ownerID,
		Name:        req.Name,
		Slug:        generated,
		Description: req.Description,
		AvatarURL:   req.AvatarURL,
		IsPublic:    isPublic,
	}
	if err := s.guildRepo.CreateGuild(ctx, guild); err != nil {
		return nil, err
	}

	created, err := s.guildRepo.FindGuildByID(ctx, guild.ID)
	if err != nil {
		return nil, err
	}

	resp := GuildToResponse(created)
	isMember := true
	role := GuildRoleOwner
	resp.IsMember = &isMember
	resp.MyRole = &role
	return resp, nil
}

func (s *guildService) GetGuildBySlug(ctx context.Context, guildSlug, viewerID string) (*GuildResponse, error) {
	guild, err := s.guildRepo.FindGuildBySlug(ctx, guildSlug)
	if err != nil {
		return nil, err
	}

	member, err := s.lookupMembership(ctx, guild.ID, viewerID)
	if err != nil {
		return nil, err
	}

	// A private guild is invisible to non-members: report it as missing rather
	// than forbidden so that its existence does not leak.
	if !guild.IsPublic && member == nil {
		return nil, ErrGuildNotFound
	}

	resp := GuildToResponse(guild)
	applyMembership(resp, member, viewerID)
	return resp, nil
}

func (s *guildService) ListGuilds(ctx context.Context, search string, limit, offset int) ([]*GuildResponse, int64, error) {
	guilds, total, err := s.guildRepo.ListPublicGuilds(ctx, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return toGuildResponses(guilds), total, nil
}

func (s *guildService) ListMyGuilds(ctx context.Context, userID string, limit, offset int) ([]*GuildResponse, int64, error) {
	guilds, total, err := s.guildRepo.ListGuildsByMember(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return toGuildResponses(guilds), total, nil
}

func (s *guildService) UpdateGuild(ctx context.Context, guildSlug, userID string, req *UpdateGuildRequest) (*GuildResponse, error) {
	if err := rejectSpamText(optionalText(req.Name), optionalText(req.Description)); err != nil {
		return nil, err
	}

	guild, err := s.guildRepo.FindGuildBySlug(ctx, guildSlug)
	if err != nil {
		return nil, err
	}

	member, err := s.lookupMembership(ctx, guild.ID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil || (member.Role != GuildRoleOwner && member.Role != GuildRoleAdmin) {
		return nil, ErrGuildNotOwned
	}

	updates := make(map[string]any)
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.AvatarURL != nil {
		updates["avatar_url"] = *req.AvatarURL
	}
	if req.IsPublic != nil {
		updates["is_public"] = *req.IsPublic
	}
	// The slug is deliberately immutable: it is the public URL of the guild.
	if err := s.guildRepo.UpdateGuild(ctx, guild.ID, updates); err != nil {
		return nil, err
	}

	updated, err := s.guildRepo.FindGuildByID(ctx, guild.ID)
	if err != nil {
		return nil, err
	}
	resp := GuildToResponse(updated)
	applyMembership(resp, member, userID)
	return resp, nil
}

func (s *guildService) DeleteGuild(ctx context.Context, guildSlug, userID string) error {
	guild, err := s.guildRepo.FindGuildBySlug(ctx, guildSlug)
	if err != nil {
		return err
	}
	if guild.OwnerID != userID {
		return ErrGuildNotOwned
	}
	return s.guildRepo.DeleteGuild(ctx, guild.ID)
}

func (s *guildService) JoinGuild(ctx context.Context, guildSlug, userID string) (*GuildMemberResponse, error) {
	guild, err := s.guildRepo.FindGuildBySlug(ctx, guildSlug)
	if err != nil {
		return nil, err
	}
	// Private guilds have no open join path yet; hide them the same way the
	// detail endpoint does.
	if !guild.IsPublic {
		return nil, ErrGuildNotFound
	}

	existing, err := s.lookupMembership(ctx, guild.ID, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrAlreadyGuildMember
	}

	member := &GuildMember{
		GuildID: guild.ID,
		UserID:  userID,
		Role:    GuildRoleMember,
	}
	if err := s.guildRepo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	created, err := s.guildRepo.FindMember(ctx, guild.ID, userID)
	if err != nil {
		return nil, err
	}
	return GuildMemberToResponse(created), nil
}

func (s *guildService) LeaveGuild(ctx context.Context, guildSlug, userID string) error {
	guild, err := s.guildRepo.FindGuildBySlug(ctx, guildSlug)
	if err != nil {
		return err
	}

	member, err := s.guildRepo.FindMember(ctx, guild.ID, userID)
	if err != nil {
		return err
	}
	// Letting the owner leave would strand the guild with nobody able to
	// administer it.
	if member.Role == GuildRoleOwner {
		return ErrGuildOwnerCannotLeave
	}
	return s.guildRepo.RemoveMember(ctx, guild.ID, userID)
}

func (s *guildService) ListMembers(ctx context.Context, guildSlug, viewerID string, limit, offset int) ([]*GuildMemberResponse, int64, error) {
	guild, err := s.guildRepo.FindGuildBySlug(ctx, guildSlug)
	if err != nil {
		return nil, 0, err
	}

	if !guild.IsPublic {
		member, err := s.lookupMembership(ctx, guild.ID, viewerID)
		if err != nil {
			return nil, 0, err
		}
		if member == nil {
			return nil, 0, ErrGuildNotFound
		}
	}

	members, total, err := s.guildRepo.ListMembers(ctx, guild.ID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	out := make([]*GuildMemberResponse, 0, len(members))
	for _, m := range members {
		out = append(out, GuildMemberToResponse(m))
	}
	return out, total, nil
}

// lookupMembership returns nil (and no error) when userID is empty or the user
// is not a member, so callers can treat "anonymous" and "not joined" alike.
func (s *guildService) lookupMembership(ctx context.Context, guildID, userID string) (*GuildMember, error) {
	if userID == "" {
		return nil, nil
	}
	member, err := s.guildRepo.FindMember(ctx, guildID, userID)
	if err != nil {
		if errors.Is(err, ErrNotGuildMember) {
			return nil, nil
		}
		return nil, err
	}
	return member, nil
}

// applyMembership fills the viewer-relative fields. They stay absent for
// anonymous callers rather than being reported as false.
func applyMembership(resp *GuildResponse, member *GuildMember, viewerID string) {
	if resp == nil || viewerID == "" {
		return
	}
	isMember := member != nil
	resp.IsMember = &isMember
	if member != nil {
		role := member.Role
		resp.MyRole = &role
	}
}

func toGuildResponses(guilds []*Guild) []*GuildResponse {
	out := make([]*GuildResponse, 0, len(guilds))
	for _, g := range guilds {
		out = append(out, GuildToResponse(g))
	}
	return out
}
