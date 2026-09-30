package model

import "time"

const (
	PostReportStatusPending   = "pending"
	PostReportStatusResolved  = "resolved"
	PostReportStatusDismissed = "dismissed"

	ModerationActionHide    = "hide"
	ModerationActionUnhide  = "unhide"
	ModerationActionDismiss = "dismiss"
)

// PostReport is one user's report of one post. At most one report per
// (post, reporter) can be pending at a time (partial unique index).
type PostReport struct {
	ID         string    `json:"id" gorm:"type:uuid;primaryKey;default:uuidv7()"`
	PostID     string    `json:"post_id" gorm:"type:uuid;not null"`
	ReporterID string    `json:"reporter_id" gorm:"type:uuid;not null"`
	Reason     string    `json:"reason" gorm:"type:varchar(32);not null"`
	Details    *string   `json:"details"`
	Status     string    `json:"status" gorm:"type:varchar(16);not null;default:pending"`
	ActionID   *string   `json:"action_id" gorm:"type:uuid"`
	CreatedAt  time.Time `json:"created_at"`

	Reporter *User                 `gorm:"foreignKey:ReporterID" json:"reporter,omitempty"`
	Action   *PostModerationAction `gorm:"foreignKey:ActionID" json:"action,omitempty"`
}

func (PostReport) TableName() string {
	return "post_reports"
}

// PostModerationAction is the audit record of one admin decision on a post.
// PostID and PostAuthorID are nulled when the post or author is deleted.
type PostModerationAction struct {
	ID           string    `json:"id" gorm:"type:uuid;primaryKey;default:uuidv7()"`
	PostID       *string   `json:"post_id" gorm:"type:uuid"`
	PostAuthorID *string   `json:"post_author_id" gorm:"type:uuid"`
	AdminID      *string   `json:"admin_id" gorm:"type:uuid"`
	Action       string    `json:"action" gorm:"type:varchar(16);not null"`
	Note         *string   `json:"note"`
	CreatedAt    time.Time `json:"created_at"`

	Admin *User `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
}

func (PostModerationAction) TableName() string {
	return "post_moderation_actions"
}
