package portfolio

type ExchangeRateResponse struct {
	From      string  `json:"from"`
	To        string  `json:"to"`
	Symbol    string  `json:"symbol"`
	Rate      float64 `json:"rate"`
	Source    string  `json:"source"`
	Cached    bool    `json:"cached"`
	FetchedAt string  `json:"fetchedAt"`
}
