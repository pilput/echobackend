package model

import (
	"time"
)

// CorporateAction is a single dividend or RUPS event persisted from the IDX
// corporate actions calendar.
type CorporateAction struct {
	ID        int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	Symbol    string     `json:"symbol" gorm:"type:text;not null"`
	Name      string     `json:"name" gorm:"type:text"`
	Type      string     `json:"type" gorm:"type:text;not null"` // "dividend" | "rups"
	EventDate time.Time  `json:"event_date" gorm:"column:event_date;type:date;not null"`
	CumDate   *time.Time `json:"cum_date" gorm:"type:date"`
	RecDate   *time.Time `json:"rec_date" gorm:"type:date"`
	PayDate   *time.Time `json:"pay_date" gorm:"type:date"`
	Amount    *float64   `json:"amount" gorm:"type:numeric(18,4)"`
	Currency  string     `json:"currency" gorm:"type:text"`
	Note      string     `json:"note" gorm:"type:text"`
	Market    string     `json:"market" gorm:"type:text;not null;default:IDX"`
	CreatedAt time.Time  `json:"created_at" gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"type:timestamptz;not null;default:now()"`
}

func (CorporateAction) TableName() string {
	return "corporate_actions"
}
