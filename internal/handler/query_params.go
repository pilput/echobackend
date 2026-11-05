package handler

import (
	"strings"
	"time"
)

// naiveFilterDateLayouts are the zone-less timestamp layouts accepted by
// parseFilterDate; they are read as UTC. Fractional seconds are optional.
var naiveFilterDateLayouts = []string{
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
}

// filterDateOnlyLayout is a YYYY-MM-DD date whose month and day may be
// unpadded ("2024-1-2"), matching chrono's %Y-%m-%d in axumbackend.
const filterDateOnlyLayout = "2006-1-2"

// rfc3339Separator replaces a space or lowercase 't' date/time separator with
// 'T', as chrono's parse_from_rfc3339 accepts both (e.g.
// "2024-01-02 10:30:00+07:00").
func rfc3339Separator(value string) string {
	if len(value) > 10 && (value[10] == ' ' || value[10] == 't') {
		return value[:10] + "T" + value[11:]
	}
	return value
}

// parseFilterDate validates a list-filter date query param (start_date,
// end_date). It accepts an RFC 3339 timestamp, a zone-less timestamp (UTC)
// or a plain YYYY-MM-DD date (UTC midnight; month/day may be unpadded). It returns the normalized UTC
// RFC 3339 value ("" when the param is empty or blank) and false when the
// value cannot be parsed, so a malformed value never reaches SQL.
func parseFilterDate(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", true
	}
	if t, err := time.Parse(time.RFC3339, rfc3339Separator(value)); err == nil {
		return t.UTC().Format(time.RFC3339Nano), true
	}
	for _, layout := range naiveFilterDateLayouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC().Format(time.RFC3339Nano), true
		}
	}
	if t, err := time.Parse(filterDateOnlyLayout, value); err == nil {
		return t.UTC().Format(time.RFC3339Nano), true
	}
	return "", false
}

// parseStatsDate returns the trimmed YYYY-MM-DD value of a stats date query
// param (startDate, endDate; month/day may be unpadded), or "" when it is empty or unparsable: stats
// endpoints ignore a bad date instead of rejecting the request.
func parseStatsDate(raw string) string {
	value := strings.TrimSpace(raw)
	t, err := time.Parse(filterDateOnlyLayout, value)
	if err != nil {
		return ""
	}
	return t.Format(time.DateOnly)
}
