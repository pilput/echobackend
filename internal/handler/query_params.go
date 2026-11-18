package handler

import (
	"strings"
	"time"
)

// filterDateOnlyLayout is a YYYY-MM-DD date whose month and day may be
// unpadded ("2024-1-2").
const filterDateOnlyLayout = "2006-1-2"

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
