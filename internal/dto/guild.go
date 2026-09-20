package dto

import (
	"echobackend/internal/model"
	"time"
)

type CreateGuildRequest struct {
	Name        string  `json:"name" validate:"required,min=3,max=100"`
	Slug        *string `json:"slug" validate:"omitempty,min=3,max=100"`
	Description *string `json:"description" validate:"omitempty,max=2000"`
	AvatarURL   *string `json:"avatar_url" validate:"omitempty,url,max=2048"`
	IsPublic    *bool   `json:"is_public"`
}

type UpdateGuildRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=3,max=100"`
	Description *string `json:"description" validate:"omitempty,max=2000"`
	AvatarURL   *string `json:"avatar_url" validate:"omitempty,url,max=2048"`
	IsPublic    *bool   `json:"is_public"`
}

type GuildResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description *string    `json:"description"`
	AvatarURL   *string    `json:"avatar_url"`
	IsPublic    bool       `json:"is_public"`
	MemberCount int64      `json:"member_count"`
	OwnerID     string     `json:"owner_id"`
	Owner       *UserBrief `json:"owner,omitempty"`
	// IsMember and MyRole are only set for authenticated callers.
	IsMember  *bool      `json:"is_member,omitempty"`
	MyRole    *string    `json:"my_role,omitempty"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

type GuildMemberResponse struct {
	ID        string     `json:"id"`
	GuildID   string     `json:"guild_id"`
	UserID    string     `json:"user_id"`
	Role      string     `json:"role"`
	User      *UserBrief `json:"user,omitempty"`
	CreatedAt *time.Time `json:"created_at"`
}

func GuildToResponse(g *model.Guild) *GuildResponse {
	if g == nil {
		return nil
	}
	return &GuildResponse{
		ID:          g.ID,
		Name:        g.Name,
		Slug:        g.Slug,
		Description: g.Description,
		AvatarURL:   g.AvatarURL,
		IsPublic:    g.IsPublic,
		MemberCount: g.MemberCount,
		OwnerID:     g.OwnerID,
		Owner:       UserToBrief(g.Owner),
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
}

func GuildMemberToResponse(m *model.GuildMember) *GuildMemberResponse {
	if m == nil {
		return nil
	}
	return &GuildMemberResponse{
		ID:        m.ID,
		GuildID:   m.GuildID,
		UserID:    m.UserID,
		Role:      m.Role,
		User:      UserToBrief(m.User),
		CreatedAt: m.CreatedAt,
	}
}
