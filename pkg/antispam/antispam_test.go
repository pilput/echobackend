package antispam

import "testing"

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
	body := "A genuine tutorial. Read user.id, java.io.File and logger.info, then see Node.js, ASP.NET and README.md."
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

func TestPostRejectReason_LinkLimit(t *testing.T) {
	for _, body := range []string{
		"See https://go.dev/doc",
		"See [docs](https://go.dev/doc) and www.example.com",
		"See https://a.example/x, and github.com",
	} {
		if reason := PostRejectReason("Title", body, nil); reason != "" {
			t.Errorf("body %q: PostRejectReason() = %q, want empty at or under the limit", body, reason)
		}
	}
	for _, body := range []string{
		"https://a.example https://b.example https://c.example",
		"[a](https://a.example) www.b.example c.example.com",
		`<a href="https://a.example">a</a> mailto:x@b.example t.me/c`,
		"https://a.example,https://b.example,https://c.example",
	} {
		if reason := PostRejectReason("Title", body, nil); reason == "" {
			t.Errorf("body %q: PostRejectReason() = empty, want link-limit rejection", body)
		}
	}
}

func TestPostRejectReason_LinksCountAcrossFields(t *testing.T) {
	if reason := PostRejectReason("https://a.example", "https://b.example", []string{"c.example.com"}); reason == "" {
		t.Error("PostRejectReason() = empty, want rejection for 3 links across title, body and tags")
	}
}

func TestPostRejectReason_ShortenerRejectedEvenWithinLimit(t *testing.T) {
	if reason := PostRejectReason("Title", "Klik bit.ly/abc123", nil); reason == "" {
		t.Error("PostRejectReason() = empty, want shortener rejection")
	}
}

func TestContainsLink_NoFalsePositives(t *testing.T) {
	for _, text := range []string{
		"Plain text without links",
		"Reading user.id and req.params.id",
		"import java.io.File; logger.info(\"x\")",
		"Node.js, Vue.js, ASP.NET and .NET Core",
		"Edit README.md and main.go",
		"Version 1.2.3 released.",
		"Visit example.community or the awww.thing page",
		"e.g. this, i.e. that. Done.Thanks",
		"Meet at 10.30 in the time slot",
		"Orbit.ly/ mission log",
		"",
	} {
		if ContainsLink(text) {
			t.Errorf("ContainsLink(%q) = true, want false", text)
		}
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
