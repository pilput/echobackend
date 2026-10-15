// Package antispam holds the shared heuristics that keep marketing spam out
// of public feeds. The same rules are mirrored in axumbackend
// (src/antispam.rs); keep the two in sync when changing thresholds.
package antispam

import (
	"fmt"
	"strconv"
	"strings"
)

// PromoTagNames marks posts written primarily to sell something. Trending
// excludes these so marketing posts don't crowd out editorial content.
// Authors should still tag honestly: the report queue treats a missing promo
// tag on an obvious ad as an aggravating signal, not a loophole.
var PromoTagNames = []string{"promo", "iklan", "ads", "advertisement", "sponsored"}

// MaxExternalLinksPerPost caps outbound links in a single post body. Link
// stuffing is the most common shape of marketing spam; legitimate authors
// can put extra links in their profile instead.
const MaxExternalLinksPerPost = 5

// bannedPostKeywords are phrases that almost never appear in legitimate
// posts but are pervasive in gambling/fraud spam. Single generic words
// (e.g. "slot") are deliberately NOT listed to avoid false positives like
// "time slot".
var bannedPostKeywords = []string{
	"togel",
	"gacor",
	"maxwin",
	"judol",
	"pinjol",
	"judi online",
	"judi slot",
	"slot online",
	"scatter hitam",
	"mahjong ways",
	"gates of olympus",
	"pragmatic play",
	"rtp live",
	"rtp slot",
	"pinjaman online",
}

// IsPromoTag reports whether a tag marks a post as promotional
// (case-insensitive, surrounding whitespace ignored).
func IsPromoTag(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, promo := range PromoTagNames {
		if lower == promo {
			return true
		}
	}
	return false
}

// CountExternalLinks counts outbound links in a post body. Both raw URLs and
// HTML/markdown links contain a scheme, so counting schemes covers every
// markup the editor can produce.
func CountExternalLinks(body string) int {
	lower := strings.ToLower(body)
	return strings.Count(lower, "http://") + strings.Count(lower, "https://")
}

// BannedKeyword returns the first banned phrase found in the title or body,
// or "" when the text is clean. Matching is case-insensitive.
func BannedKeyword(title, body string) string {
	haystack := strings.ToLower(title + "\n" + body)
	for _, keyword := range bannedPostKeywords {
		if strings.Contains(haystack, keyword) {
			return keyword
		}
	}
	return ""
}

// RejectReason returns "" when a post looks fine, otherwise a short
// user-facing reason explaining why it was rejected. Callers surface the
// reason through the 422 envelope so authors learn the rule instead of just
// hitting a wall.
func RejectReason(title, body string) string {
	if keyword := BannedKeyword(title, body); keyword != "" {
		return "contains banned keyword " + strconv.Quote(keyword) +
			". Gambling/fraud spam is not allowed; honest promotions must use the " +
			strconv.Quote("promo") + " tag and follow the community guidelines"
	}

	if links := CountExternalLinks(body); links > MaxExternalLinksPerPost {
		return fmt.Sprintf(
			"too many external links (%d, max %d). Put extra links in your profile or tag the post \"promo\" instead of stuffing links into the body",
			links,
			MaxExternalLinksPerPost,
		)
	}

	return ""
}
