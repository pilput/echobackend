package dto

import (
	"errors"
	"time"

	"echobackend/internal/model"
)

type UserBrief struct {
	ID       string  `json:"id"`
	Username *string `json:"username"`
	Image    *string `json:"image"`
}

func UserToBrief(u *model.User) *UserBrief {
	if u == nil || u.ID == "" {
		return nil
	}
	return &UserBrief{
		ID:       u.ID,
		Username: u.Username,
		Image:    u.Image,
	}
}

type UserResponse struct {
	ID             string         `json:"id"`
	Email          string         `json:"email,omitempty"`
	Name           string         `json:"name"`
	Username       *string        `json:"username"`
	Image          *string        `json:"image"`
	FirstName      *string        `json:"first_name"`
	LastName       *string        `json:"last_name"`
	FollowersCount int64          `json:"followers_count"`
	FollowingCount int64          `json:"following_count"`
	IsFollowing    *bool          `json:"is_following,omitempty"`
	IsSuperAdmin   *bool          `json:"is_super_admin,omitempty"`
	Profile        *model.Profile `json:"profile,omitempty"`
	CreatedAt      *time.Time     `json:"created_at"`
	UpdatedAt      *time.Time     `json:"updated_at"`
	DeletedAt      *time.Time     `json:"deleted_at,omitempty"`
	LastLoggedAt   *time.Time     `json:"last_logged_at,omitempty"`
}

type CurrentUserResponse struct {
	ID             string         `json:"id"`
	Email          string         `json:"email"`
	Name           string         `json:"name"`
	Username       *string        `json:"username"`
	Image          *string        `json:"image"`
	FirstName      *string        `json:"first_name"`
	LastName       *string        `json:"last_name"`
	IsSuperAdmin   *bool          `json:"is_super_admin"`
	FollowersCount int64          `json:"followers_count"`
	FollowingCount int64          `json:"following_count"`
	Profile        *model.Profile `json:"profile,omitempty"`
	CreatedAt      *time.Time     `json:"created_at"`
	UpdatedAt      *time.Time     `json:"updated_at"`
}

type PublicUserResponse struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Username       *string        `json:"username"`
	Image          *string        `json:"image"`
	FirstName      *string        `json:"first_name"`
	LastName       *string        `json:"last_name"`
	FollowersCount int64          `json:"followers_count"`
	FollowingCount int64          `json:"following_count"`
	IsFollowing    *bool          `json:"is_following,omitempty"`
	Profile        *model.Profile `json:"profile,omitempty"`
	CreatedAt      *time.Time     `json:"created_at"`
	UpdatedAt      *time.Time     `json:"updated_at"`
}

// fullName builds the display name from the optional first/last name: both
// set -> "first last", only one set -> that one, neither -> "". An empty-string
// pointer still counts as set.
func fullName(first, last *string) string {
	switch {
	case first != nil && last != nil:
		return *first + " " + *last
	case first != nil:
		return *first
	case last != nil:
		return *last
	default:
		return ""
	}
}

func UserToResponse(u *model.User) *UserResponse {
	if u == nil {
		return nil
	}
	return &UserResponse{
		ID:             u.ID,
		Name:           fullName(u.FirstName, u.LastName),
		Username:       u.Username,
		Image:          u.Image,
		FirstName:      u.FirstName,
		LastName:       u.LastName,
		FollowersCount: u.FollowersCount,
		FollowingCount: u.FollowingCount,
		Profile:        u.Profile,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

func UserToAdminResponse(u *model.User) *UserResponse {
	resp := UserToResponse(u)
	if resp != nil {
		resp.Email = u.Email
		resp.IsSuperAdmin = u.IsSuperAdmin
		resp.LastLoggedAt = u.LastLoggedAt
		if u.DeletedAt.Valid {
			t := u.DeletedAt.Time
			resp.DeletedAt = &t
		}
	}
	return resp
}

func UserToPublicResponse(u *model.User) *PublicUserResponse {
	if u == nil {
		return nil
	}
	return &PublicUserResponse{
		ID:             u.ID,
		Name:           fullName(u.FirstName, u.LastName),
		Username:       u.Username,
		Image:          u.Image,
		FirstName:      u.FirstName,
		LastName:       u.LastName,
		FollowersCount: u.FollowersCount,
		FollowingCount: u.FollowingCount,
		Profile:        u.Profile,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

func UserToCurrentUserResponse(u *model.User) *CurrentUserResponse {
	if u == nil {
		return nil
	}
	return &CurrentUserResponse{
		ID:             u.ID,
		Email:          u.Email,
		Name:           fullName(u.FirstName, u.LastName),
		Username:       u.Username,
		Image:          u.Image,
		FirstName:      u.FirstName,
		LastName:       u.LastName,
		IsSuperAdmin:   u.IsSuperAdmin,
		FollowersCount: u.FollowersCount,
		FollowingCount: u.FollowingCount,
		Profile:        u.Profile,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

type CreateUserRequest struct {
	Username  string `json:"username" validate:"required,min=3,max=30"`
	Email     string `json:"email" validate:"required,email,max=255"`
	Password  string `json:"password" validate:"required,min=8,max=128"`
	FirstName string `json:"first_name" validate:"omitempty,max=100"`
	LastName  string `json:"last_name" validate:"omitempty,max=100"`
}

// UpdateUserRequest is the admin update body. first_name/last_name are
// nullable: omitted or null stores NULL, a string (even "") stores it.
// is_super_admin is optional: omitted or null keeps the stored value.
type UpdateUserRequest struct {
	FirstName    *string `json:"first_name" validate:"omitempty,max=100"`
	LastName     *string `json:"last_name" validate:"omitempty,max=100"`
	Username     string  `json:"username" validate:"required,min=3,max=30"`
	Email        string  `json:"email" validate:"required,email,max=255"`
	IsSuperAdmin *bool   `json:"is_super_admin"`
}

type UserDeletedFilter string

const (
	UserDeletedFilterActive UserDeletedFilter = ""
	UserDeletedFilterOnly   UserDeletedFilter = "true"
	UserDeletedFilterAll    UserDeletedFilter = "all"
)

func ParseUserDeletedFilter(value string) (UserDeletedFilter, error) {
	switch value {
	case "", "false":
		return UserDeletedFilterActive, nil
	case string(UserDeletedFilterOnly):
		return UserDeletedFilterOnly, nil
	case string(UserDeletedFilterAll):
		return UserDeletedFilterAll, nil
	default:
		return "", errors.New("deleted must be true, false, or all")
	}
}

type UserListOrder string

const (
	UserListOrderCreatedAt  UserListOrder = "created_at"
	UserListOrderLastLogged UserListOrder = "last_logged_at"
	UserListOrderUsername   UserListOrder = "username"
	UserListOrderEmail      UserListOrder = "email"
	UserListOrderDefault    UserListOrder = UserListOrderCreatedAt
)

func ParseUserListOrder(value string) UserListOrder {
	switch UserListOrder(value) {
	case UserListOrderLastLogged, UserListOrderUsername, UserListOrderEmail:
		return UserListOrder(value)
	default:
		return UserListOrderDefault
	}
}

func ParseUserOrderDesc(value string) bool {
	return value != "asc"
}
