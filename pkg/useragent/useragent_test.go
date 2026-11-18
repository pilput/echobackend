package useragent

import "testing"

func TestIsLikelyBot_DetectsAutomatedClients(t *testing.T) {
	for _, ua := range []string{
		"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
		"facebookexternalhit/1.1",
		"Mozilla/5.0 HeadlessChrome/120.0",
		"curl/8.5.0",
		"python-requests/2.32",
		"",
		"   ",
	} {
		if !IsLikelyBot(ua) {
			t.Errorf("IsLikelyBot(%q) = false, want true", ua)
		}
	}
}

func TestIsLikelyBot_BrowsersAreNotBots(t *testing.T) {
	for _, ua := range []string{
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/131.0 Safari/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) Version/18.0 Mobile Safari/604.1",
	} {
		if IsLikelyBot(ua) {
			t.Errorf("IsLikelyBot(%q) = true, want false", ua)
		}
	}
}
