package dto

// CorporateActionResponse represents a single corporate action event
// returned by the calendar endpoint.
type CorporateActionResponse struct {
	Symbol   string   `json:"symbol"`
	Name     string   `json:"name,omitempty"`
	Type     string   `json:"type"` // "dividend" | "rups"
	Date     string   `json:"date"` // ISO 8601: "2025-07-15"
	CumDate  *string  `json:"cum_date,omitempty"`
	RecDate  *string  `json:"rec_date,omitempty"`
	PayDate  *string  `json:"pay_date,omitempty"`
	Amount   *float64 `json:"amount,omitempty"` // dividend per share
	Currency string   `json:"currency,omitempty"`
	Note     string   `json:"note,omitempty"`
	Market   string   `json:"market"` // "IDX" | "US"
}

// CorporateActionCalendarResponse is the top-level response for GET /holdings/calendar.
type CorporateActionCalendarResponse struct {
	From    string                    `json:"from"`
	To      string                    `json:"to"`
	Total   int                       `json:"total"`
	Cached  bool                      `json:"cached"`
	Actions []CorporateActionResponse `json:"actions"`
}
