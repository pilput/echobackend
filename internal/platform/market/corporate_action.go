package market

import (
	"context"
	"time"
)

// CorporateActionType represents the type of a corporate action event.
type CorporateActionType string

const (
	// ActionDividend represents a cash dividend event.
	ActionDividend CorporateActionType = "dividend"
	// ActionRUPS represents a General Meeting of Shareholders (Rapat Umum Pemegang Saham).
	ActionRUPS CorporateActionType = "rups"
)

// CorporateAction holds the details of a single corporate action event.
type CorporateAction struct {
	// Symbol is the stock ticker (e.g. "BBCA", "AAPL").
	Symbol string
	// Name is the company's display name.
	Name string
	// Type is the kind of corporate action (dividend, rups, etc.).
	Type CorporateActionType
	// Date is the primary event date (ex-date for dividends, meeting date for RUPS).
	Date time.Time
	// CumDate is the last trading date to still be entitled to the dividend
	// (cum-date, one exchange day before the ex-date). Nil for non-dividend events.
	CumDate *time.Time
	// RecDate is the recording/entitlement date used to determine the
	// shareholder register (nil for non-dividend events).
	RecDate *time.Time
	// PayDate is the dividend payment date (nil for non-dividend events).
	PayDate *time.Time
	// Amount is the gross dividend amount per share (nil for non-dividend events).
	Amount *float64
	// Currency is the currency of the dividend amount (e.g. "IDR").
	Currency string
	// Note contains additional information about the event.
	Note string
	// Market identifies the exchange ("IDX" or "US").
	Market string
}

// CorporateActionClient fetches corporate action events.
type CorporateActionClient interface {
	// GetCorporateActions returns all corporate actions within the [from, to] date range.
	// Implementations must be fail-open: errors should be skipped, not propagated.
	GetCorporateActions(ctx context.Context, from, to time.Time) ([]CorporateAction, error)
}
