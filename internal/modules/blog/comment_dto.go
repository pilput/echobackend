package blog

import (
	"echobackend/internal/core/userref"
	"time"
)

type CreateCommentRequest struct {
	Text string `json:"text" validate:"required,min=1,max=1000"`
}

type CommentResponse struct {
	ID              string         `json:"id"`
	PostID          string         `json:"post_id"`
	ParentCommentID *string        `json:"parent_comment_id,omitempty"`
	Text            string         `json:"text"`
	User            *userref.Brief `json:"user,omitempty"`
	CreatedAt       *time.Time     `json:"created_at"`
	UpdatedAt       *time.Time     `json:"updated_at"`
}

func CommentToResponse(pc *PostComment) *CommentResponse {
	if pc == nil {
		return nil
	}
	userResp := userref.ToBrief(pc.User)
	return &CommentResponse{
		ID:              pc.ID,
		PostID:          pc.PostID,
		ParentCommentID: pc.ParentCommentID,
		Text:            pc.Text,
		User:            userResp,
		CreatedAt:       pc.CreatedAt,
		UpdatedAt:       pc.UpdatedAt,
	}
}
