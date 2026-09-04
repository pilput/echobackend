package dto

import (
	"echobackend/internal/model"
	"time"
)

type CreatePostRequest struct {
	Title     string   `json:"title" validate:"required,min=7"`
	PhotoURL  string   `json:"photo_url"`
	Slug      string   `json:"slug" validate:"required,min=7"`
	Body      string   `json:"body" validate:"required,min=10"`
	Published bool     `json:"published"`
	Tags      []string `json:"tags"`
}

type UpdatePostRequest struct {
	Title     string   `json:"title"`
	PhotoURL  string   `json:"photo_url"`
	Slug      string   `json:"slug"`
	Body      string   `json:"body"`
	Published *bool    `json:"published"`
	Tags      []string `json:"tags"`
}

type PostQueryFilter struct {
	Limit     int      `json:"limit" query:"limit"`
	Offset    int      `json:"offset" query:"offset"`
	Search    string   `json:"search" query:"search"`
	SortBy    string   `json:"sort_by" query:"sort_by"`
	SortOrder string   `json:"sort_order" query:"sort_order"`
	StartDate string   `json:"start_date" query:"start_date"`
	EndDate   string   `json:"end_date" query:"end_date"`
	Published *bool    `json:"published" query:"published"`
	CreatedBy string   `json:"created_by" query:"created_by"`
	Tags      []string `json:"tags" query:"tags"`
}

func (f *PostQueryFilter) ValidSortFields() map[string]string {
	return map[string]string{
		"id":         "posts.id",
		"title":      "posts.title",
		"created_at": "posts.created_at",
		"updated_at": "posts.updated_at",
		"view_count": "posts.view_count",
		"like_count": "posts.like_count",
	}
}

func (f *PostQueryFilter) ValidSortOrders() []string {
	return []string{"asc", "desc"}
}

func (f *PostQueryFilter) GetSortField() string {
	if field, exists := f.ValidSortFields()[f.SortBy]; exists {
		return field
	}
	return "posts.created_at"
}

func (f *PostQueryFilter) GetSortOrder() string {
	for _, order := range f.ValidSortOrders() {
		if order == f.SortOrder {
			return order
		}
	}
	return "desc"
}

type PostResponse struct {
	ID            string        `json:"id"`
	Title         *string       `json:"title"`
	PhotoURL      *string       `json:"photo_url"`
	Body          *string       `json:"body"`
	Slug          *string       `json:"slug"`
	ViewCount     int64         `json:"view_count"`
	LikeCount     int64         `json:"like_count"`
	BookmarkCount int64         `json:"bookmark_count"`
	Published     *bool         `json:"published"`
	PublishedAt   *time.Time    `json:"published_at"`
	User          *UserBrief    `json:"user,omitempty"`
	Tags          []TagResponse `json:"tags,omitempty"`
	CreatedAt     *time.Time    `json:"created_at"`
	UpdatedAt     *time.Time    `json:"updated_at"`
}

func PostToResponse(p *model.Post) *PostResponse {
	if p == nil {
		return nil
	}
	userResp := UserToBrief(p.User)

	var tagResponses []TagResponse
	if p.Tags != nil {
		tagResponses = make([]TagResponse, len(p.Tags))
		for i, tag := range p.Tags {
			tagResponses[i] = *TagToResponse(&tag)
		}
	}

	return &PostResponse{
		ID:            p.ID,
		Title:         p.Title,
		PhotoURL:      p.PhotoURL,
		Body:          p.Body,
		Slug:          p.Slug,
		ViewCount:     p.ViewCount,
		LikeCount:     p.LikeCount,
		BookmarkCount: p.BookmarkCount,
		Published:     p.Published,
		PublishedAt:   p.PublishedAt,
		User:          userResp,
		Tags:          tagResponses,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

// TruncatePostBodies truncates the Body of each post to maxRunes runes.
// Safe for multi-byte UTF-8 characters.
func TruncatePostBodies(posts []*PostResponse, maxRunes int) {
	for _, post := range posts {
		if post.Body == nil {
			continue
		}
		r := []rune(*post.Body)
		if len(r) > maxRunes {
			truncated := string(r[:maxRunes]) + " ..."
			post.Body = &truncated
		}
	}
}

type SitemapPost struct {
	Username  *string    `json:"username"`
	Slug      *string    `json:"slug"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}
