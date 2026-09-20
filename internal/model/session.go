package model

import (
	"time"
)

// Session is one link in a refresh-token rotation chain.
//
// Every token minted from a single login shares a FamilyID. Refreshing rotates
// the chain forward: the current row is stamped with RotatedAt/ReplacedBy and a
// successor row is inserted. Rotated rows are deliberately kept until they
// expire, because a replayed token has to stay recognisable for long enough to
// revoke its family (RFC 9700 §4.14.2).
//
// RefreshToken stores the SHA-256 hash of the token, never the token itself.
type Session struct {
	ID           string `json:"id" gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID     string `json:"family_id" gorm:"type:uuid;not null;index"`
	RefreshToken string `json:"-" gorm:"type:text;not null;uniqueIndex"`
	UserID       string `json:"user_id" gorm:"type:uuid;not null"`

	UserAgent *string `json:"user_agent"`
	IPAddress *string `json:"ip_address" gorm:"type:varchar(45)"`

	CreatedAt time.Time `json:"created_at" gorm:"not null;default:now()"`
	// ExpiresAt is the sliding inactivity deadline: each rotation pushes it
	// forward, so an idle chain dies while an active one does not.
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	// AbsoluteExpiresAt caps the whole family regardless of activity. It is
	// copied unchanged from the root session, so rotation can never extend it.
	AbsoluteExpiresAt time.Time `json:"absolute_expires_at" gorm:"not null"`

	// RotatedAt is set when this token has been exchanged for a successor.
	// A rotated token presented after the grace window is a replay.
	RotatedAt *time.Time `json:"rotated_at"`
	// ReplacedBy is the hashed successor token, kept for audit only.
	ReplacedBy *string `json:"-" gorm:"type:text"`

	User *User `json:"-" gorm:"foreignKey:UserID"`
}

func (Session) TableName() string {
	return "sessions"
}

// Rotated reports whether this token has already been exchanged.
func (s *Session) Rotated() bool {
	return s.RotatedAt != nil
}
