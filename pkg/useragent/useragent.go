// Package useragent classifies HTTP User-Agent strings.
package useragent

import "strings"

// botMarkers are lowercase User-Agent fragments of automated clients. Extend
// this list when new crawlers show up in post_views.
var botMarkers = []string{
	// generic crawler words (Googlebot, bingbot, AhrefsBot, Slurp, ...)
	"bot",
	"crawler",
	"spider",
	"slurp",
	// link previews (Slack, Discord, WhatsApp, Facebook, embed services)
	"preview",
	"facebookexternalhit",
	"embedly",
	// headless browsers and auditing tools
	"headless",
	"lighthouse",
	// CLI and library HTTP clients
	"curl/",
	"wget/",
	"python-requests",
	"httpclient",
}

// IsLikelyBot reports whether a request most likely comes from an automated
// client (crawler, link-preview fetcher, headless browser, CLI tool) rather
// than a person. An empty User-Agent counts as automated: every browser sends
// one. Use it to keep such traffic out of metrics; it is a heuristic, not a
// security boundary.
func IsLikelyBot(userAgent string) bool {
	if strings.TrimSpace(userAgent) == "" {
		return true
	}
	ua := strings.ToLower(userAgent)
	for _, marker := range botMarkers {
		if strings.Contains(ua, marker) {
			return true
		}
	}
	return false
}
