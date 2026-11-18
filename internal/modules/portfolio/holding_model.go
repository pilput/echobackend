package portfolio

import (
	"time"
)

type HoldingType struct {
	ID    int     `json:"id" gorm:"primaryKey;autoIncrement"`
	Code  string  `json:"code" gorm:"uniqueIndex;type:varchar(50);not null"`
	Name  string  `json:"name" gorm:"type:varchar(100);not null"`
	Notes *string `json:"notes"`
}

func (HoldingType) TableName() string {
	return "holding_types"
}

type Holding struct {
	ID             int64        `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID         string       `json:"user_id" gorm:"type:uuid;not null;index:idx_holdings_user"`
	Name           string       `json:"name" gorm:"type:text;not null"`
	Symbol         *string      `json:"symbol" gorm:"type:text"`
	Platform       string       `json:"platform" gorm:"type:text;not null"`
	HoldingTypeID  int          `json:"holding_type_id" gorm:"type:smallint;not null;index:idx_holdings_holding_type_id"`
	HoldingType    *HoldingType `json:"holding_type" gorm:"foreignKey:HoldingTypeID;references:ID"`
	Currency       string       `json:"currency" gorm:"type:char(3);not null"`
	InvestedAmount string       `json:"invested_amount" gorm:"type:numeric(18,2);not null;default:0"`
	CurrentValue   string       `json:"current_value" gorm:"type:numeric(18,2);not null;default:0"`
	GainAmount     string       `json:"gain_amount" gorm:"type:numeric(18,2);<-:false"`
	GainPercent    string       `json:"gain_percent" gorm:"type:numeric(18,2);<-:false"`
	Units          *string      `json:"units" gorm:"type:numeric(24,3)"`
	AvgBuyPrice    *string      `json:"avg_buy_price" gorm:"type:numeric(18,8)"`
	CurrentPrice   *string      `json:"current_price" gorm:"type:numeric(18,8)"`
	LastUpdated    *time.Time   `json:"last_updated" gorm:"type:timestamptz"`
	Notes          *string      `json:"notes" gorm:"type:text"`
	CreatedAt      time.Time    `json:"created_at" gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt      time.Time    `json:"updated_at" gorm:"type:timestamptz;not null;default:now()"`
	Month          int          `json:"month" gorm:"type:int;not null;default:1;index:idx_holdings_month_year"`
	Year           int          `json:"year" gorm:"type:int;not null;default:2000;index:idx_holdings_month_year"`
}

func (Holding) TableName() string {
	return "holdings"
}
