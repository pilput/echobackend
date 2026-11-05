package handler

import "testing"

func TestParseFilterDate(t *testing.T) {
	tests := []struct {
		raw    string
		want   string
		wantOK bool
	}{
		{"", "", true},
		{"   ", "", true},
		{"2024-01-02", "2024-01-02T00:00:00Z", true},
		{" 2024-01-02 ", "2024-01-02T00:00:00Z", true},
		{"2024-01-02T10:30:00+07:00", "2024-01-02T03:30:00Z", true},
		{"2024-01-02T10:30:00.5Z", "2024-01-02T10:30:00.5Z", true},
		{"2024-01-02 10:30:00", "2024-01-02T10:30:00Z", true},
		{"2024-01-02T10:30:00", "2024-01-02T10:30:00Z", true},
		{"2024-01-02 10:30:00+07:00", "2024-01-02T03:30:00Z", true},
		{"2024-01-02t10:30:00Z", "2024-01-02T10:30:00Z", true},
		{"2024-1-2", "2024-01-02T00:00:00Z", true},
		{"not-a-date", "", false},
		{"2024-13-01", "", false},
		{"02/01/2024", "", false},
	}
	for _, tt := range tests {
		got, ok := parseFilterDate(tt.raw)
		if ok != tt.wantOK || got != tt.want {
			t.Errorf("parseFilterDate(%q) = (%q, %v), want (%q, %v)", tt.raw, got, ok, tt.want, tt.wantOK)
		}
	}
}

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
