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

// MaxLinksPerPost caps links in a single post (title, body and tags
// together). Link stuffing is the most common shape of marketing spam;
// legitimate authors can put extra links in their profile instead.
const MaxLinksPerPost = 2

// bannedKeywords are phrases that almost never appear in legitimate content
// but are pervasive in gambling/fraud spam. Single generic words (e.g.
// "slot") are deliberately NOT listed to avoid false positives like
// "time slot". Entries must be lowercase ASCII words separated by single
// spaces, the form normalize produces.
var bannedKeywords = []string{
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

// BannedKeyword returns the first banned phrase found in any of texts, or ""
// when they are clean. Texts are normalized first, so case, homoglyphs,
// letter-spacing and leetspeak don't hide a keyword. Multi-word phrases also
// match when written as one word ("judionline").
func BannedKeyword(texts ...string) string {
	haystack := normalize(strings.Join(texts, "\n"))
	for _, keyword := range bannedKeywords {
		if strings.Contains(haystack, keyword) ||
			strings.Contains(haystack, strings.ReplaceAll(keyword, " ", "")) {
			return keyword
		}
	}
	return ""
}

// TextRejectReason returns "" when user-written text (comments, guild names
// and descriptions) looks fine, otherwise a short user-facing reason.
func TextRejectReason(texts ...string) string {
	if keyword := BannedKeyword(texts...); keyword != "" {
		return keywordReason(keyword)
	}
	for _, text := range texts {
		if reason := BlockedLink(text); reason != "" {
			return reason
		}
	}
	return ""
}

func keywordReason(keyword string) string {
	return "contains banned keyword " + strconv.Quote(keyword) +
		". Gambling/fraud spam is not allowed; honest promotions must use the " +
		strconv.Quote("promo") + " tag and follow the community guidelines"
}

// PostRejectReason returns "" when a post looks fine, otherwise a short
// user-facing reason explaining why it was rejected. Tags go through the
// same keyword and link checks as the title and body. Callers surface the
// reason through the 422 envelope so authors learn the rule instead of just
// hitting a wall.
func PostRejectReason(title, body string, tags []string) string {
	texts := append([]string{title, body}, tags...)
	if reason := TextRejectReason(texts...); reason != "" {
		return reason
	}

	if links := CountLinks(texts...); links > MaxLinksPerPost {
		return fmt.Sprintf(
			"too many links (%d, max %d). Put extra links in your profile instead of stuffing them into the post",
			links,
			MaxLinksPerPost,
		)
	}

	return ""
}
