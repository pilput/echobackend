package market

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	rapidAPIQuoteHost         = "yh-finance.p.rapidapi.com"
	rapidAPIQuoteBaseURL      = "https://yh-finance.p.rapidapi.com"
	maxSymbolsPerQuoteRequest = 50
)

// QuoteClient fetches market quotes for symbols.
type QuoteClient interface {
	GetQuotes(ctx context.Context, symbols []string) (map[string]float64, error)
}

// RapidAPIQuoteClient fetches real-time market prices using RapidAPI.
type RapidAPIQuoteClient struct {
	httpClient *http.Client
	apiKey     string
	host       string
	baseURL    string
}

// NewRapidAPIQuoteClient creates a new RapidAPI quote client using YH Finance.
func NewRapidAPIQuoteClient(apiKey string, httpClient *http.Client) *RapidAPIQuoteClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &RapidAPIQuoteClient{
		httpClient: httpClient,
		apiKey:     apiKey,
		host:       rapidAPIQuoteHost,
		baseURL:    rapidAPIQuoteBaseURL,
	}
}

// GetQuotes fetches prices for a slice of symbols in batch requests.
func (c *RapidAPIQuoteClient) GetQuotes(ctx context.Context, symbols []string) (map[string]float64, error) {
	normalized := NormalizeSymbols(symbols)
	if len(normalized) == 0 {
		return map[string]float64{}, nil
	}

	quotes := make(map[string]float64, len(normalized))
	for start := 0; start < len(normalized); start += maxSymbolsPerQuoteRequest {
		end := min(start+maxSymbolsPerQuoteRequest, len(normalized))

		batchQuotes, err := c.fetchBatch(ctx, normalized[start:end])
		if err != nil {
			return nil, err
		}
		maps.Copy(quotes, batchQuotes)
	}

	return quotes, nil
}

func (c *RapidAPIQuoteClient) fetchBatch(ctx context.Context, symbols []string) (map[string]float64, error) {
	query := url.Values{}
	symbolsParam := strings.Join(symbols, ",")
	query.Set("symbols", symbolsParam)
	query.Set("ticker", symbolsParam)

	endpoint := "/market/v2/get-quotes"
	if strings.Contains(c.host, "yahoo-finance15") {
		endpoint = "/api/v1/markets/stock/quotes"
	}
	reqURL := c.baseURL + endpoint + "?" + query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create rapidapi quote request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("X-RapidAPI-Key", c.apiKey)
	}
	if c.host != "" {
		req.Header.Set("X-RapidAPI-Host", c.host)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (pilput-backend)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request rapidapi quotes: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read rapidapi quote response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rapidapi quote returned status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	return parseQuoteResponse(body)
}

// NormalizeSymbols trims, uppercases, and deduplicates a slice of ticker symbols.
func NormalizeSymbols(symbols []string) []string {
	seen := make(map[string]struct{}, len(symbols))
	normalized := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		symbol = strings.ToUpper(strings.TrimSpace(symbol))
		if symbol == "" {
			continue
		}
		if _, ok := seen[symbol]; ok {
			continue
		}
		seen[symbol] = struct{}{}
		normalized = append(normalized, symbol)
	}
	return normalized
}

// parseQuoteResponse handles various JSON envelopes commonly used by RapidAPI quote providers:
// 1. {"body": [{"symbol": "AAPL", "regularMarketPrice": 150.0}, ...]}
// 2. {"quoteResponse": {"result": [{"symbol": "AAPL", "regularMarketPrice": 150.0}, ...]}}
// 3. [{"symbol": "AAPL", "regularMarketPrice": 150.0}, ...]
// 4. {"spark": {"result": [{"symbol": "AAPL", "response": [{"meta": {"regularMarketPrice": 150.0}}]}]}}
func parseQuoteResponse(body []byte) (map[string]float64, error) {
	quotes := make(map[string]float64)

	// Try format 1: {"body": [...]}
	var bodyWrapper struct {
		Body []quoteItem `json:"body"`
	}
	if err := json.Unmarshal(body, &bodyWrapper); err == nil && len(bodyWrapper.Body) > 0 {
		for _, item := range bodyWrapper.Body {
			extractQuoteItem(item, quotes)
		}
		return quotes, nil
	}

	// Try format 2: {"quoteResponse": {"result": [...]}}
	var qrWrapper struct {
		QuoteResponse struct {
			Result []quoteItem `json:"result"`
		} `json:"quoteResponse"`
	}
	if err := json.Unmarshal(body, &qrWrapper); err == nil && len(qrWrapper.QuoteResponse.Result) > 0 {
		for _, item := range qrWrapper.QuoteResponse.Result {
			extractQuoteItem(item, quotes)
		}
		return quotes, nil
	}

	// Try format 3: Direct slice `[{"symbol": ...}]`
	var sliceWrapper []quoteItem
	if err := json.Unmarshal(body, &sliceWrapper); err == nil && len(sliceWrapper) > 0 {
		for _, item := range sliceWrapper {
			extractQuoteItem(item, quotes)
		}
		return quotes, nil
	}

	// Try format 4: Spark format
	var sparkWrapper struct {
		Spark struct {
			Result []struct {
				Symbol   string `json:"symbol"`
				Response []struct {
					Meta struct {
						RegularMarketPrice float64 `json:"regularMarketPrice"`
					} `json:"meta"`
				} `json:"response"`
			} `json:"result"`
		} `json:"spark"`
	}
	if err := json.Unmarshal(body, &sparkWrapper); err == nil && len(sparkWrapper.Spark.Result) > 0 {
		for _, res := range sparkWrapper.Spark.Result {
			if len(res.Response) > 0 {
				price := res.Response[0].Meta.RegularMarketPrice
				if price > 0 {
					quotes[strings.ToUpper(strings.TrimSpace(res.Symbol))] = price
				}
			}
		}
		return quotes, nil
	}

	return quotes, nil
}

type quoteItem struct {
	Symbol             string          `json:"symbol"`
	RegularMarketPrice json.RawMessage `json:"regularMarketPrice"`
	Price              json.RawMessage `json:"price"`
	Close              json.RawMessage `json:"close"`
}

func extractQuoteItem(item quoteItem, quotes map[string]float64) {
	sym := strings.ToUpper(strings.TrimSpace(item.Symbol))
	if sym == "" {
		return
	}

	for _, raw := range [][]byte{item.RegularMarketPrice, item.Price, item.Close} {
		if len(raw) == 0 {
			continue
		}
		if price, ok := parseRawPrice(raw); ok && price > 0 {
			quotes[sym] = price
			return
		}
	}
}

func parseRawPrice(raw []byte) (float64, bool) {
	// Try float number
	var num float64
	if err := json.Unmarshal(raw, &num); err == nil {
		return num, true
	}

	// Try string number
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		var val float64
		if _, err := fmt.Sscanf(str, "%f", &val); err == nil {
			return val, true
		}
	}

	// Try object with "raw" field: {"raw": 150.0, "fmt": "150.00"}
	var obj struct {
		Raw float64 `json:"raw"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.Raw > 0 {
		return obj.Raw, true
	}

	return 0, false
}
