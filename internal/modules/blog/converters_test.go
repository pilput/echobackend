package blog

import (
	"echobackend/internal/core/userref"
	"encoding/json"
	"strings"
	"testing"
	"time"
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

	filter.SortBy = "bookmark_count"
	if got := filter.GetSortField(); got != "posts.bookmark_count" {
		t.Fatalf("expected bookmark_count sort field, got %q", got)
	}
	filter.SortBy = "posts.id; DROP TABLE posts"
	filter.SortOrder = "sideways"
	if got := filter.GetSortField(); got != "posts.created_at" {
		t.Fatalf("expected unknown sort field to fall back, got %q", got)
	}
	if got := filter.GetSortOrder(); got != "desc" {
		t.Fatalf("expected unknown sort order to fall back, got %q", got)
	}
}

func TestPostToResponse(t *testing.T) {
	if PostToResponse(nil) != nil {
		t.Fatal("expected nil response for nil post")
	}

	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	post := &Post{
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
		User: &userref.Ref{
			ID:       "user-1",
			Username: new("author"),
		},
		Tags: []Tag{{ID: 1, Name: "go"}, {ID: 2, Name: "api"}},
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
	user := &userref.Ref{ID: "user-1", Username: new("commenter")}
	parentID := "parent-1"
	comment := CommentToResponse(&PostComment{
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

	like := PostLikeToResponse(&PostLike{ID: "like-1", PostID: "post-1", UserID: "user-1", User: user, CreatedAt: &now})
	if like.ID != "like-1" || like.User == nil || like.User.Username != user.Username {
		t.Fatalf("unexpected like response: %+v", like)
	}

	ip := "127.0.0.1"
	agent := "test-agent"
	view := PostViewToResponse(&PostView{ID: "view-1", PostID: "post-1", UserID: new("user-1"), IPAddress: &ip, UserAgent: &agent, CreatedAt: &now, UpdatedAt: &now})
	if view.ID != "view-1" || view.UserID == nil || *view.UserID != "user-1" {
		t.Fatalf("unexpected view response: %+v", view)
	}
	// Viewer IP address and user agent must never be exposed in API responses.
	if raw, err := json.Marshal(view); err != nil || strings.Contains(string(raw), ip) || strings.Contains(string(raw), agent) {
		t.Fatalf("view response leaks viewer metadata: %s (err=%v)", raw, err)
	}

	tag := TagToResponse(&Tag{ID: 1, Name: "golang"})
	if tag.ID != 1 || tag.Name != "golang" {
		t.Fatalf("unexpected tag response: %+v", tag)
	}
}

// TestFullName mirrors axum's test_full_name: a lone first or last name is
// used as is, and an empty-string pointer still counts as set.
func TestBookmarkFolderToResponse_BookmarkCount(t *testing.T) {
	preloaded := &BookmarkFolder{ID: "f1", PostBookmarks: []PostBookmark{{ID: "b1"}, {ID: "b2"}}}
	if got := BookmarkFolderToResponse(preloaded).BookmarkCount; got != 2 {
		t.Fatalf("preloaded count = %d, want 2", got)
	}

	// A folder nested in a bookmark has no PostBookmarks preloaded; the
	// repository supplies the COUNT instead.
	n := int64(7)
	counted := &BookmarkFolder{ID: "f2", BookmarkCount: &n}
	resp := BookmarkToResponse(&PostBookmark{ID: "b3", Folder: counted})
	if resp.Folder == nil || resp.Folder.BookmarkCount != 7 {
		t.Fatalf("nested folder = %+v, want bookmark_count 7", resp.Folder)
	}
}
