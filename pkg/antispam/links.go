package antispam

import "strings"

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

// CountExternalLinks counts outbound links in a post body. Both raw URLs and
// HTML/markdown links contain a scheme, so counting schemes covers every
// markup the editor can produce.
func CountExternalLinks(body string) int {
	lower := strings.ToLower(body)
	return strings.Count(lower, "http://") + strings.Count(lower, "https://")
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
