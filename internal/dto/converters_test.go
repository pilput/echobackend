package dto

import (
	"testing"
	"time"

	"echobackend/internal/model"

	"gorm.io/gorm"
)

func TestPostQueryFilterSortDefaultsAndValidValues(t *testing.T) {
	filter := &PostQueryFilter{}
	if got := filter.GetSortField(); got != "posts.created_at" {
		t.Fatalf("expected default sort field posts.created_at, got %q", got)
	}
	if got := filter.GetSortOrder(); got != "desc" {
		t.Fatalf("expected default sort order desc, got %q", got)
	}

	filter.SortBy = "like_count"
	filter.SortOrder = "asc"
	if got := filter.GetSortField(); got != "posts.like_count" {
		t.Fatalf("expected like_count sort field, got %q", got)
	}
	if got := filter.GetSortOrder(); got != "asc" {
		t.Fatalf("expected asc sort order, got %q", got)
	}
}

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

func TestPostToResponse(t *testing.T) {
	if PostToResponse(nil) != nil {
		t.Fatal("expected nil response for nil post")
	}

	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	post := &model.Post{
		ID:            "post-1",
		Title:         new("Post title"),
		PhotoURL:      new("https://example.com/photo.png"),
		Body:          new("Post body"),
		Slug:          new("post-title"),
		ViewCount:     11,
		LikeCount:     5,
		BookmarkCount: 2,
		Published:     new(true),
		PublishedAt:   &now,
		CreatedAt:     &now,
		UpdatedAt:     &now,
		User: &model.User{
			ID:       "user-1",
			Username: new("author"),
		},
		Tags: []model.Tag{{ID: 1, Name: "go"}, {ID: 2, Name: "api"}},
	}

	resp := PostToResponse(post)
	if resp.ID != post.ID || resp.Title != post.Title || resp.ViewCount != 11 || resp.LikeCount != 5 || resp.BookmarkCount != 2 {
		t.Fatalf("unexpected post response: %+v", resp)
	}
	if resp.User == nil || resp.User.ID != "user-1" {
		t.Fatalf("expected user brief, got %+v", resp.User)
	}
	if len(resp.Tags) != 2 || resp.Tags[0].Name != "go" || resp.Tags[1].Name != "api" {
		t.Fatalf("unexpected tags: %+v", resp.Tags)
	}
}

func TestTruncatePostBodies(t *testing.T) {
	short := "short"
	long := "こんにちは世界"
	posts := []*PostResponse{
		{ID: "nil-body"},
		{ID: "short", Body: &short},
		{ID: "long", Body: &long},
	}

	TruncatePostBodies(posts, 5)

	if posts[0].Body != nil {
		t.Fatalf("expected nil body to remain nil, got %q", *posts[0].Body)
	}
	if *posts[1].Body != "short" {
		t.Fatalf("expected short body unchanged, got %q", *posts[1].Body)
	}
	if *posts[2].Body != "こんにちは ..." {
		t.Fatalf("expected multibyte-safe truncation, got %q", *posts[2].Body)
	}
}

func TestSimpleModelConverters(t *testing.T) {
	if TagToResponse(nil) != nil || CommentToResponse(nil) != nil || PostLikeToResponse(nil) != nil || PostViewToResponse(nil) != nil {
		t.Fatal("expected nil converter inputs to return nil")
	}

	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	user := &model.User{ID: "user-1", Username: new("commenter")}
	parentID := "parent-1"
	comment := CommentToResponse(&model.PostComment{
		ID:              "comment-1",
		PostID:          "post-1",
		ParentCommentID: &parentID,
		Text:            "hello",
		User:            user,
		CreatedAt:       &now,
		UpdatedAt:       &now,
	})
	if comment.ID != "comment-1" || comment.ParentCommentID != &parentID || comment.User == nil || comment.User.ID != "user-1" {
		t.Fatalf("unexpected comment response: %+v", comment)
	}

	like := PostLikeToResponse(&model.PostLike{ID: "like-1", PostID: "post-1", UserID: "user-1", User: user, CreatedAt: &now})
	if like.ID != "like-1" || like.User == nil || like.User.Username != user.Username {
		t.Fatalf("unexpected like response: %+v", like)
	}

	ip := "127.0.0.1"
	agent := "test-agent"
	view := PostViewToResponse(&model.PostView{ID: "view-1", PostID: "post-1", UserID: new("user-1"), IPAddress: &ip, UserAgent: &agent, CreatedAt: &now, UpdatedAt: &now})
	if view.ID != "view-1" || view.UserID == nil || *view.UserID != "user-1" || view.IPAddress != &ip || view.UserAgent != &agent {
		t.Fatalf("unexpected view response: %+v", view)
	}

	tag := TagToResponse(&model.Tag{ID: 1, Name: "golang"})
	if tag.ID != 1 || tag.Name != "golang" {
		t.Fatalf("unexpected tag response: %+v", tag)
	}
}
