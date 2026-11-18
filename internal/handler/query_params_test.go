package handler

import "testing"

func TestParseStatsDate(t *testing.T) {
	tests := map[string]string{
		"":                     "",
		"2024-01-02":           "2024-01-02",
		" 2024-01-02 ":         "2024-01-02",
		"2024-01-02T00:00:00Z": "",
		"garbage":              "",
		"2024-02-30":           "",
		"2024-1-2":             "2024-01-02",
		"2024-01-02 10:30:00":  "",
	}
	for raw, want := range tests {
		if got := parseStatsDate(raw); got != want {
			t.Errorf("parseStatsDate(%q) = %q, want %q", raw, got, want)
		}
	}
}
