package dto

import (
	"time"

	"echobackend/internal/model"
)

// CreatePostReportRequest is the body of POST /posts/:id/reports. The reason
// list mirrors chk_post_reports_reason; keep the two in sync.
type CreatePostReportRequest struct {
	Reason  string  `json:"reason" validate:"required,oneof=spam harassment hate nsfw misinformation other"`
	Details *string `json:"details" validate:"omitempty,max=1000"`
}

// ModeratePostRequest is the body of POST /posts/:id/moderation. The action
// list mirrors chk_post_moderation_actions_action.
type ModeratePostRequest struct {
	Action string  `json:"action" validate:"required,oneof=hide unhide dismiss"`
	Note   *string `json:"note" validate:"omitempty,max=1000"`
}

// PostReportGroup is one row of the admin queue: a post with its reports
// collapsed into a count, so a moderator triages posts rather than reports.
type PostReportGroup struct {
	PostID         string     `json:"post_id"`
	PostTitle      *string    `json:"post_title"`
	PostSlug       *string    `json:"post_slug"`
	AuthorUsername *string    `json:"author_username"`
	HiddenAt       *time.Time `json:"hidden_at"`
	ReportCount    int64      `json:"report_count"`
	TopReason      string     `json:"top_reason"`
	LastReportedAt time.Time  `json:"last_reported_at"`
}

type ModerationActionResponse struct {
	ID        string     `json:"id"`
	PostID    *string    `json:"post_id"`
	Action    string     `json:"action"`
	Note      *string    `json:"note"`
	CreatedAt time.Time  `json:"created_at"`
	Admin     *UserBrief `json:"admin,omitempty"`
}

func ModerationActionToResponse(a *model.PostModerationAction) *ModerationActionResponse {
	if a == nil {
		return nil
	}
	return &ModerationActionResponse{
		ID:        a.ID,
		PostID:    a.PostID,
		Action:    a.Action,
		Note:      a.Note,
		CreatedAt: a.CreatedAt,
		Admin:     UserToBrief(a.Admin),
	}
}

type PostReportResponse struct {
	ID        string     `json:"id"`
	PostID    string     `json:"post_id"`
	Reason    string     `json:"reason"`
	Details   *string    `json:"details"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	Reporter  *UserBrief `json:"reporter,omitempty"`
	// Resolution is the admin decision that closed the report; absent while pending.
	Resolution *ModerationActionResponse `json:"resolution,omitempty"`
}

func PostReportToResponse(r *model.PostReport) *PostReportResponse {
	if r == nil {
		return nil
	}
	return &PostReportResponse{
		ID:         r.ID,
		PostID:     r.PostID,
		Reason:     r.Reason,
		Details:    r.Details,
		Status:     r.Status,
		CreatedAt:  r.CreatedAt,
		Reporter:   UserToBrief(r.Reporter),
		Resolution: ModerationActionToResponse(r.Action),
	}
}
