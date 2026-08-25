package market

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRapidAPIQuoteClient_GetQuotes_BodyFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-RapidAPI-Key") != "test-key" {
			t.Errorf("expected X-RapidAPI-Key 'test-key', got %q", r.Header.Get("X-RapidAPI-Key"))
		}
		if r.Header.Get("X-RapidAPI-Host") != "test-host" {
			t.Errorf("expected X-RapidAPI-Host 'test-host', got %q", r.Header.Get("X-RapidAPI-Host"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"body": [
				{"symbol": "BBCA.JK", "regularMarketPrice": 9850},
				{"symbol": "AAPL", "regularMarketPrice": {"raw": 180.5, "fmt": "180.50"}},
				{"symbol": "USDIDR=X", "regularMarketPrice": "16200.5"}
			]
		}`))
	}))
	defer server.Close()

	client := NewRapidAPIQuoteClient("test-key", "test-host", server.URL, server.Client())
	quotes, err := client.GetQuotes(context.Background(), []string{"bbca.jk", "AAPL", "USDIDR=X"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if quotes["BBCA.JK"] != 9850 {
		t.Errorf("expected BBCA.JK = 9850, got %v", quotes["BBCA.JK"])
	}
	if quotes["AAPL"] != 180.5 {
		t.Errorf("expected AAPL = 180.5, got %v", quotes["AAPL"])
	}
	if quotes["USDIDR=X"] != 16200.5 {
		t.Errorf("expected USDIDR=X = 16200.5, got %v", quotes["USDIDR=X"])
	}
}

func TestRapidAPIQuoteClient_GetQuotes_QuoteResponseFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"quoteResponse": {
				"result": [
					{"symbol": "BBRI.JK", "regularMarketPrice": 5200}
				]
			}
		}`))
	}))
	defer server.Close()

	client := NewRapidAPIQuoteClient("test-key", "test-host", server.URL, server.Client())
	quotes, err := client.GetQuotes(context.Background(), []string{"BBRI.JK"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if quotes["BBRI.JK"] != 5200 {
		t.Errorf("expected BBRI.JK = 5200, got %v", quotes["BBRI.JK"])
	}
}

func TestRapidAPIQuoteClient_GetQuotes_EmptySymbols(t *testing.T) {
	client := NewRapidAPIQuoteClient("test-key", "test-host", "http://dummy", nil)
	quotes, err := client.GetQuotes(context.Background(), []string{"", "  "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(quotes) != 0 {
		t.Errorf("expected empty map, got %v", quotes)
	}
}

func TestNormalizeSymbols(t *testing.T) {
	symbols := []string{" bbca.jk ", "AAPL", "BBCA.JK", "", "  ", "aapl"}
	normalized := NormalizeSymbols(symbols)

	if len(normalized) != 2 {
		t.Fatalf("expected 2 unique symbols, got %d: %v", len(normalized), normalized)
	}
	if normalized[0] != "BBCA.JK" || normalized[1] != "AAPL" {
		t.Errorf("unexpected normalized slice: %v", normalized)
	}
}
