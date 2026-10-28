package antispam

import (
	"strings"
	"unicode"
)

// linkShorteners hide a link's destination, which is exactly what gambling
// and fraud spam relies on. Authors can always paste the full URL instead.
var linkShorteners = []string{
	"adf.ly",
	"bit.ly",
	"cutt.ly",
	"is.gd",
	"ow.ly",
	"rb.gy",
	"rebrand.ly",
	"s.id",
	"shorturl.at",
	"t.ly",
	"tinyurl.com",
	"v.gd",
}

// gamblingHostWords only appear in hostnames of gambling sites. "slot" is
// handled separately because legitimate scheduling sites use it too.
var gamblingHostWords = []string{"casino", "judi", "poker", "sbobet"}

// linkTLDs are top-level domains that flag a scheme-less "example.com" as a
// link. Deliberately short: TLDs that double as common code identifiers
// (".id", ".io", ".info", ".app", ".link", ".store") would reject posts that
// merely mention user.id or java.io.
var linkTLDs = map[string]struct{}{
	"com": {}, "org": {}, "xyz": {}, "biz": {}, "club": {}, "vip": {},
	"co": {}, "bet": {}, "casino": {},
}

// bareLinkDomains are messenger/link-in-bio hosts that spam pastes without a
// scheme and that the TLD list cannot catch ("t.me/promo").
var bareLinkDomains = []string{"t.me", "wa.me", "linktr.ee"}

// ContainsLink reports whether text contains a link in any form authors
// paste: a URL with a scheme, "www." hosts, mailto: addresses, a bare
// "example.com" domain, or a scheme-less messenger link such as "t.me/x".
// Markdown and HTML links always carry one of these, so they are covered.
func ContainsLink(text string) bool {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "://") || strings.Contains(lower, "mailto:") {
		return true
	}
	if hasWWWHost(lower) || hasBareDomain(lower) {
		return true
	}
	for _, domain := range bareLinkDomains {
		if containsBareLink(lower, domain) {
			return true
		}
	}
	return false
}

// CountLinks counts links across texts. Text is split into words at
// whitespace, brackets and quotes, so markdown, HTML and plain links each
// count once; a word glued from several URLs counts once per scheme.
func CountLinks(texts ...string) int {
	count := 0
	for _, text := range texts {
		words := strings.FieldsFunc(text, func(r rune) bool {
			return unicode.IsSpace(r) || strings.ContainsRune(`()[]<>"'`, r)
		})
		for _, word := range words {
			if ContainsLink(word) {
				count += max(1, strings.Count(word, "://"))
			}
		}
	}
	return count
}

// hasWWWHost matches "www." at the start of a word, not the tail of one
// ("awww.").
func hasWWWHost(lower string) bool {
	for offset := 0; ; {
		i := strings.Index(lower[offset:], "www.")
		if i < 0 {
			return false
		}
		start := offset + i
		if start == 0 || !isASCIIAlnum(rune(lower[start-1])) {
			return true
		}
		offset = start + len("www.")
	}
}

// hasBareDomain matches "label.tld" for the TLDs in linkTLDs, requiring the
// TLD to end the word so "example.community" does not match ".co".
func hasBareDomain(lower string) bool {
	for offset := 0; ; {
		i := strings.IndexByte(lower[offset:], '.')
		if i < 0 {
			return false
		}
		dot := offset + i
		offset = dot + 1
		if dot == 0 || !isASCIIAlnum(rune(lower[dot-1])) {
			continue
		}
		rest := lower[dot+1:]
		end := 0
		for end < len(rest) && rest[end] >= 'a' && rest[end] <= 'z' {
			end++
		}
		if _, ok := linkTLDs[rest[:end]]; !ok {
			continue
		}
		if end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
			continue
		}
		return true
	}
}

// BlockedLink returns a user-facing reason for the first link that points to
// a link shortener or a gambling-looking host, or "" when every link is fine.
func BlockedLink(text string) string {
	lower := strings.ToLower(text)
	for _, host := range linkHosts(lower) {
		if shortener := matchShortener(host); shortener != "" {
			return shortenerReason(shortener)
		}
		if isGamblingHost(host) {
			return "links to gambling sites are not allowed (" + host + ")"
		}
	}
	// Shortened links are often pasted without a scheme ("bit.ly/xyz").
	for _, shortener := range linkShorteners {
		if containsBareLink(lower, shortener) {
			return shortenerReason(shortener)
		}
	}
	return ""
}

func shortenerReason(shortener string) string {
	return "link shortener " + shortener +
		" is not allowed; paste the full destination URL so readers can see where it goes"
}

// linkHosts extracts the host of every http(s) URL in already-lowercased
// text, without a "www." prefix or port.
func linkHosts(lower string) []string {
	var hosts []string
	rest := lower
	for {
		i := strings.Index(rest, "://")
		if i < 0 {
			return hosts
		}
		scheme := rest[:i]
		rest = rest[i+len("://"):]
		if !strings.HasSuffix(scheme, "http") && !strings.HasSuffix(scheme, "https") {
			continue
		}
		end := strings.IndexFunc(rest, isHostTerminator)
		if end < 0 {
			end = len(rest)
		}
		if host := strings.TrimPrefix(rest[:end], "www."); host != "" {
			hosts = append(hosts, host)
		}
	}
}

func isHostTerminator(r rune) bool {
	return !(isASCIIAlnum(r) || r == '.' || r == '-')
}

func matchShortener(host string) string {
	for _, shortener := range linkShorteners {
		if host == shortener || strings.HasSuffix(host, "."+shortener) {
			return shortener
		}
	}
	return ""
}

func isGamblingHost(host string) bool {
	for _, word := range gamblingHostWords {
		if strings.Contains(host, word) {
			return true
		}
	}
	return hasSlotNumber(host)
}

// hasSlotNumber matches the "slot88" / "slot777" naming pattern of gambling
// sites while leaving hosts like "timeslot.app" alone.
func hasSlotNumber(host string) bool {
	for rest := host; ; {
		i := strings.Index(rest, "slot")
		if i < 0 {
			return false
		}
		rest = rest[i+len("slot"):]
		if rest != "" && rest[0] >= '0' && rest[0] <= '9' {
			return true
		}
	}
}

// containsBareLink reports whether domain appears as the start of a link
// path ("bit.ly/abc"), not as the tail of a longer word ("orbit.ly/").
func containsBareLink(lower, domain string) bool {
	needle := domain + "/"
	for offset := 0; ; {
		i := strings.Index(lower[offset:], needle)
		if i < 0 {
			return false
		}
		start := offset + i
		if start == 0 || isHostTerminator(rune(lower[start-1])) {
			return true
		}
		offset = start + len(needle)
	}
}
