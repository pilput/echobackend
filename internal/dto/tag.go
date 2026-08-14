package dto

import (
	"echobackend/internal/model"
	"time"
)

type TagResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type CreateTagRequest struct {
	Name string `json:"name" validate:"required,min=1,max=30"`
}

type UpdateTagRequest struct {
	Name string `json:"name" validate:"required,min=1,max=30"`
}

type TrendingTagResponse struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	TotalViews    int64  `json:"total_views"`
	TotalLikes    int64  `json:"total_likes"`
	TrendingScore int64  `json:"trending_score"`
}

func TagToResponse(t *model.Tag) *TagResponse {
	if t == nil {
		return nil
	}
	return &TagResponse{
		ID:   t.ID,
		Name: t.Name,
	}
}

type SitemapTag struct {
	Name      string     `json:"name"`
	CreatedAt *time.Time `json:"created_at"`
}
