package market

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	rapidAPIIDXHost    = "indonesia-stock-exchange-idx.p.rapidapi.com"
	rapidAPIIDXBaseURL = "https://indonesia-stock-exchange-idx.p.rapidapi.com"
	rapidAPIDateFormat = "2006-01-02"
)

// RapidAPIIDXClient fetches IDX corporate actions (dividend & RUPS) from RapidAPI.
type RapidAPIIDXClient struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

// NewRapidAPIIDXClient creates a new RapidAPI IDX client.
// apiKey is the X-RapidAPI-Key header value from your RapidAPI subscription.
// Pass an empty string to create a no-op client that always returns empty results.
func NewRapidAPIIDXClient(apiKey string, httpClient *http.Client) *RapidAPIIDXClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &RapidAPIIDXClient{
		httpClient: httpClient,
		apiKey:     apiKey,
		baseURL:    rapidAPIIDXBaseURL,
	}
}

// GetCorporateActions implements CorporateActionClient.
// It fetches dividend and RUPS calendars from RapidAPI IDX.
// Errors from individual endpoints are logged and skipped.
func (c *RapidAPIIDXClient) GetCorporateActions(ctx context.Context, from, to time.Time) ([]CorporateAction, error) {
	if c.apiKey == "" {
		return nil, nil
	}

	fromStr := from.Format(rapidAPIDateFormat)
	toStr := to.Format(rapidAPIDateFormat)

	var actions []CorporateAction

	// Fetch dividends. Fail-open: skip dividends on error (RUPS may still succeed).
	dividends, err := c.fetchDividends(ctx, fromStr, toStr)
	if err == nil {
		for _, d := range dividends {
			if d.CompanySymbol == "" {
				continue
			}
			sym := normalizeIDXSymbol(d.CompanySymbol)

			var exDate time.Time
			if d.ExDateStr != "" {
				if t, err := time.Parse(rapidAPIDateFormat, d.ExDateStr); err == nil {
					exDate = t
				}
			}
			if exDate.IsZero() {
				continue
			}

			action := CorporateAction{
				Symbol:   sym,
				Name:     sym, // No company name in dividends endpoint, default to ticker
				Type:     ActionDividend,
				Date:     exDate,
				Currency: normalizeCurrency(d.CurrencyStr),
				Market:   "IDX",
			}

			if d.CumDateStr != "" {
				if t, err := time.Parse(rapidAPIDateFormat, d.CumDateStr); err == nil {
					action.CumDate = &t
				}
			}

			if d.RecDateStr != "" {
				if t, err := time.Parse(rapidAPIDateFormat, d.RecDateStr); err == nil {
					action.RecDate = &t
				}
			}

			if d.PayDateStr != "" {
				if t, err := time.Parse(rapidAPIDateFormat, d.PayDateStr); err == nil {
					action.PayDate = &t
				}
			}

			if d.ValueStr != "" {
				if val, err := strconv.ParseFloat(d.ValueStr, 64); err == nil && val > 0 {
					action.Amount = &val
				}
			}

			actions = append(actions, action)
		}
	}

	// Fetch RUPS. Fail-open: skip RUPS on error (dividends may still be returned).
	rupsList, rupsErr := c.fetchRUPS(ctx, fromStr, toStr)
	if rupsErr == nil {
		for _, r := range rupsList {
			if r.CompanySymbol == "" {
				continue
			}
			sym := normalizeIDXSymbol(r.CompanySymbol)

			var meetingDate time.Time
			if r.DateStr != "" {
				if t, err := time.Parse(rapidAPIDateFormat, r.DateStr); err == nil {
					meetingDate = t
				}
			}
			if meetingDate.IsZero() {
				continue
			}

			name := r.CompanyName
			if name == "" {
				name = sym
			}

			note := ""
			if r.VenueStr != "" {
				if r.TimeStr != "" {
					note = fmt.Sprintf("Waktu: %s, Tempat: %s", r.TimeStr, r.VenueStr)
				} else {
					note = "Tempat: " + r.VenueStr
				}
			}

			actions = append(actions, CorporateAction{
				Symbol: sym,
				Name:   name,
				Type:   ActionRUPS,
				Date:   meetingDate,
				Note:   note,
				Market: "IDX",
			})
		}
	}

	return actions, nil
}

// --- internal fetch helpers ---

type rapidAPIDividendItem struct {
	CompanySymbol string `json:"company_symbol"`
	CumDateStr    string `json:"dividend_cumdate"`
	ExDateStr     string `json:"dividend_exdate"`
	RecDateStr    string `json:"dividend_recdate"`
	PayDateStr    string `json:"dividend_paydate"`
	ValueStr      string `json:"dividend_value"`
	CurrencyStr   string `json:"dividend_currency"`
}

type rapidAPIRUPSItem struct {
	CompanySymbol string `json:"company_symbol"`
	CompanyName   string `json:"company_name"`
	DateStr       string `json:"rups_date"`
	TimeStr       string `json:"rups_time"`
	VenueStr      string `json:"rups_venue"`
}

func (c *RapidAPIIDXClient) fetchDividends(ctx context.Context, from, to string) ([]rapidAPIDividendItem, error) {
	endpoint := "/api/calendar/dividend"
	params := url.Values{}
	params.Set("from", from)
	params.Set("to", to)

	body, err := c.doRequest(ctx, endpoint, params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Data struct {
				Dividend []rapidAPIDividendItem `json:"dividend"`
			} `json:"data"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode dividend response: %w", err)
	}
	return resp.Data.Data.Dividend, nil
}

func (c *RapidAPIIDXClient) fetchRUPS(ctx context.Context, from, to string) ([]rapidAPIRUPSItem, error) {
	endpoint := "/api/calendar/rups"
	params := url.Values{}
	params.Set("from", from)
	params.Set("to", to)

	body, err := c.doRequest(ctx, endpoint, params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Data struct {
				RUPS []rapidAPIRUPSItem `json:"rups"`
			} `json:"data"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode rups response: %w", err)
	}
	return resp.Data.Data.RUPS, nil
}

func (c *RapidAPIIDXClient) doRequest(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	reqURL := c.baseURL + endpoint
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request %s: %w", endpoint, err)
	}
	req.Header.Set("X-RapidAPI-Key", c.apiKey)
	req.Header.Set("X-RapidAPI-Host", rapidAPIIDXHost)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s response: %w", endpoint, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rapidapi idx %s returned status %d: %s", endpoint, resp.StatusCode, truncate(string(body), 200))
	}
	return body, nil
}

// --- helpers ---

// normalizeIDXSymbol uppercases and trims whitespace from an IDX ticker.
// Also strips the ".JK" suffix if present (Yahoo Finance format).
func normalizeIDXSymbol(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.TrimSuffix(s, ".JK")
	return s
}

func normalizeCurrency(c string) string {
	c = strings.ToUpper(strings.TrimSpace(c))
	c = strings.TrimPrefix(c, "CURRENCY_")
	if c == "" {
		return "IDR"
	}
	return c
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
