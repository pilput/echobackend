package antispam

import (
	"strings"
	"testing"
)

func TestIsPromoTag(t *testing.T) {
	for _, name := range []string{"promo", "Promo", " PROMO ", "iklan", "ads", "advertisement", "sponsored"} {
		if !IsPromoTag(name) {
			t.Errorf("IsPromoTag(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"golang", "tutorial", "promo-code", "", "ad"} {
		if IsPromoTag(name) {
			t.Errorf("IsPromoTag(%q) = true, want false", name)
		}
	}
}

func TestPostRejectReason_CleanPost(t *testing.T) {
	body := "A genuine tutorial with a [docs link](https://example.com/docs) and https://example.com/repo."
	if reason := PostRejectReason("How to write Go", body, nil); reason != "" {
		t.Errorf("PostRejectReason() = %q, want empty", reason)
	}
}

func TestPostRejectReason_TimeSlotIsNotGambling(t *testing.T) {
	// "slot" alone must not trigger: false positives erode author trust.
	body := "Pick a time slot for the meeting. The slot machine analogy does not apply here."
	if reason := PostRejectReason("Scheduling", body, nil); reason != "" {
		t.Errorf("PostRejectReason() = %q, want empty", reason)
	}
}

func TestPostRejectReason_TooManyLinks(t *testing.T) {
	var links strings.Builder
	for i := 0; i < MaxExternalLinksPerPost+1; i++ {
		links.WriteString("https://example.com/item ")
	}
	if reason := PostRejectReason("Buy now", links.String(), nil); reason == "" {
		t.Error("PostRejectReason() = empty, want link-stuffing rejection")
	}

	var atLimit strings.Builder
	for i := 0; i < MaxExternalLinksPerPost; i++ {
		atLimit.WriteString("https://example.com/item ")
	}
	if reason := PostRejectReason("Buy now", atLimit.String(), nil); reason != "" {
		t.Errorf("PostRejectReason() = %q, want empty at exactly the limit", reason)
	}
}

func TestPostRejectReason_BannedKeyword(t *testing.T) {
	cases := []struct {
		name  string
		title string
		body  string
	}{
		{"keyword in body", "Promo menarik", "Dapatkan bonus gacor hari ini"},
		{"keyword in title", "Situs togel terpercaya 2026", "Konten biasa saja"},
		{"case insensitive", "Info", "RTP LIVE 98% di sini"},
		{"multi word", "Info", "Pinjaman online cepat cair tanpa syarat"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if reason := PostRejectReason(tc.title, tc.body, nil); reason == "" {
				t.Error("PostRejectReason() = empty, want banned-keyword rejection")
			}
		})
	}
}

func TestPostRejectReason_BannedTag(t *testing.T) {
	if reason := PostRejectReason("Info", "Konten biasa", []string{"golang", "Slot Gacor"}); reason == "" {
		t.Error("PostRejectReason() = empty, want banned-keyword rejection from a tag")
	}
}

func TestBannedKeyword_ObfuscatedSpellings(t *testing.T) {
	cases := map[string]string{
		"leetspeak":          "Situs g4c0r hari ini",
		"letter spacing":     "Main di situs G A C O R sekarang",
		"dotted letters":     "Bonus t.o.g.e.l setiap hari",
		"zero-width space":   "Bonus ga\u200bcor setiap hari",
		"cyrillic homoglyph": "Bonus g\u0430cor setiap hari",
		"diacritics":         "Bonus gàcór setiap hari",
		"fullwidth":          "Bonus ｇａｃｏｒ setiap hari",
		"math bold":          "Bonus 𝐠𝐚𝐜𝐨𝐫 setiap hari",
		"symbol for letter":  "Bonus g@cor setiap hari",
		"phrase as one word": "Daftar judionline sekarang",
		"phrase over lines":  "Daftar judi\n\nonline sekarang",
		"brand with digits":  "Daftar di gacor88 sekarang",
	}
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			if keyword := BannedKeyword(text); keyword == "" {
				t.Errorf("BannedKeyword(%q) = empty, want a match", text)
			}
		})
	}
}

func TestBannedKeyword_NoFalsePositives(t *testing.T) {
	clean := []string{
		"Our mega corp acquired a startup.",            // "mega corp" must not merge into "gacor"
		"Helps the team to gel quickly.",               // "to gel" must not merge into "togel"
		"Released on 2026-10-18 with 98% coverage.",    // plain numbers are not leetspeak
		"Pick a time slot, then go online to confirm.", // generic words stay allowed
		"Café crème with a résumé attached.",
	}
	for _, text := range clean {
		if keyword := BannedKeyword(text); keyword != "" {
			t.Errorf("BannedKeyword(%q) = %q, want empty", text, keyword)
		}
	}
}

func TestBlockedLink(t *testing.T) {
	blocked := []string{
		"Klik https://bit.ly/abc123",
		"Klik bit.ly/abc123",
		"Klik [di sini](https://www.s.id/promo)",
		"Daftar https://slot88-resmi.xyz/daftar",
		"Daftar http://mega-casino.example",
		"Daftar https://sbobet.example:8080/",
	}
	for _, text := range blocked {
		if reason := BlockedLink(text); reason == "" {
			t.Errorf("BlockedLink(%q) = empty, want rejection", text)
		}
	}

	allowed := []string{
		"Docs at https://go.dev/doc and https://github.com/org/repo",
		"Book at https://timeslot.app/booking",
		"Visit orbit.ly/ for the mission log",
		"Plain text without links",
	}
	for _, text := range allowed {
		if reason := BlockedLink(text); reason != "" {
			t.Errorf("BlockedLink(%q) = %q, want empty", text, reason)
		}
	}
}

func TestTextRejectReason(t *testing.T) {
	if reason := TextRejectReason("Mantap, cek bit.ly/xyz ya"); reason == "" {
		t.Error("TextRejectReason() = empty, want shortener rejection")
	}
	if reason := TextRejectReason("Nice write-up, thanks!"); reason != "" {
		t.Errorf("TextRejectReason() = %q, want empty", reason)
	}
}
