package dto

import (
	"testing"
	"time"

	"echobackend/internal/model"

	"gorm.io/gorm"
)

func TestUserConverters(t *testing.T) {
	if UserToBrief(nil) != nil {
		t.Fatal("expected nil brief for nil user")
	}
	if UserToBrief(&model.User{}) != nil {
		t.Fatal("expected nil brief for user without ID")
	}
	if UserToResponse(nil) != nil {
		t.Fatal("expected nil response for nil user")
	}
	if UserToPublicResponse(nil) != nil {
		t.Fatal("expected nil public response for nil user")
	}
	if UserToCurrentUserResponse(nil) != nil {
		t.Fatal("expected nil current user response for nil user")
	}

	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	deletedAt := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	user := &model.User{
		ID:             "user-1",
		Email:          "user@example.com",
		FirstName:      new("Jane"),
		LastName:       new("Doe"),
		Username:       new("jdoe"),
		Image:          new("https://example.com/avatar.png"),
		IsSuperAdmin:   new(true),
		FollowersCount: 7,
		FollowingCount: 3,
		CreatedAt:      &now,
		UpdatedAt:      &now,
		DeletedAt:      gorm.DeletedAt{Time: deletedAt, Valid: true},
	}

	brief := UserToBrief(user)
	if brief.ID != user.ID || brief.Username != user.Username || brief.Image != user.Image {
		t.Fatalf("unexpected brief: %+v", brief)
	}

	resp := UserToResponse(user)
	if resp.Name != "Jane Doe" || resp.Email != "" || resp.FollowersCount != 7 || resp.DeletedAt != nil {
		t.Fatalf("unexpected user response: %+v", resp)
	}

	admin := UserToAdminResponse(user)
	if admin.IsSuperAdmin == nil || !*admin.IsSuperAdmin || admin.DeletedAt == nil || !admin.DeletedAt.Equal(deletedAt) || admin.Email != user.Email {
		t.Fatalf("unexpected admin response: %+v", admin)
	}

	public := UserToPublicResponse(user)
	if public.Name != "Jane Doe" || public.FollowingCount != 3 {
		t.Fatalf("unexpected public response: %+v", public)
	}

	current := UserToCurrentUserResponse(user)
	if current.IsSuperAdmin == nil || !*current.IsSuperAdmin || current.Name != "Jane Doe" {
		t.Fatalf("unexpected current user response: %+v", current)
	}
}

func TestFullName(t *testing.T) {
	cases := []struct {
		first, last *string
		want        string
	}{
		{new("John"), new("Doe"), "John Doe"},
		{new("John"), nil, "John"},
		{nil, new("Doe"), "Doe"},
		{nil, nil, ""},
		{new(""), new("Doe"), " Doe"},
	}
	for _, tc := range cases {
		user := &model.User{ID: "u1", FirstName: tc.first, LastName: tc.last}
		if got := fullName(tc.first, tc.last); got != tc.want {
			t.Errorf("fullName(%v, %v) = %q, want %q", tc.first, tc.last, got, tc.want)
		}
		if got := UserToResponse(user).Name; got != tc.want {
			t.Errorf("UserToResponse name = %q, want %q", got, tc.want)
		}
		if got := UserToPublicResponse(user).Name; got != tc.want {
			t.Errorf("UserToPublicResponse name = %q, want %q", got, tc.want)
		}
		if got := UserToCurrentUserResponse(user).Name; got != tc.want {
			t.Errorf("UserToCurrentUserResponse name = %q, want %q", got, tc.want)
		}
	}
}
