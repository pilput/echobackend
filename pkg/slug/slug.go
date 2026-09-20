// Package slug turns human names into URL-safe identifiers.
package slug

import (
	"strings"
	"unicode"
)

// Make converts s into a lowercase, hyphen-separated slug: runs of anything
// that is not a letter or digit collapse into a single hyphen, and leading and
// trailing hyphens are trimmed. Non-ASCII letters are kept as-is, so a name
// that is entirely non-Latin still yields a usable slug instead of an empty
// string. The result is truncated to maxLen runes (0 or less means no limit).
func Make(s string, maxLen int) string {
	var b strings.Builder
	b.Grow(len(s))

	prevHyphen := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevHyphen = false
		case !prevHyphen && b.Len() > 0:
			b.WriteRune('-')
			prevHyphen = true
		}
	}

	out := strings.Trim(b.String(), "-")
	if maxLen > 0 {
		out = truncateRunes(out, maxLen)
	}
	return strings.Trim(out, "-")
}

func truncateRunes(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
