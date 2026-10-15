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

func TestRejectReason_CleanPost(t *testing.T) {
	body := "A genuine tutorial with a [docs link](https://example.com/docs) and https://example.com/repo."
	if reason := RejectReason("How to write Go", body); reason != "" {
		t.Errorf("RejectReason() = %q, want empty", reason)
	}
}

func TestRejectReason_TimeSlotIsNotGambling(t *testing.T) {
	// "slot" alone must not trigger: false positives erode author trust.
	body := "Pick a time slot for the meeting. The slot machine analogy does not apply here."
	if reason := RejectReason("Scheduling", body); reason != "" {
		t.Errorf("RejectReason() = %q, want empty", reason)
	}
}

func TestRejectReason_TooManyLinks(t *testing.T) {
	var links strings.Builder
	for i := 0; i < MaxExternalLinksPerPost+1; i++ {
		links.WriteString("https://example.com/item ")
	}
	if reason := RejectReason("Buy now", links.String()); reason == "" {
		t.Error("RejectReason() = empty, want link-stuffing rejection")
	}

	var atLimit strings.Builder
	for i := 0; i < MaxExternalLinksPerPost; i++ {
		atLimit.WriteString("https://example.com/item ")
	}
	if reason := RejectReason("Buy now", atLimit.String()); reason != "" {
		t.Errorf("RejectReason() = %q, want empty at exactly the limit", reason)
	}
}

func TestRejectReason_BannedKeyword(t *testing.T) {
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
			if reason := RejectReason(tc.title, tc.body); reason == "" {
				t.Error("RejectReason() = empty, want banned-keyword rejection")
			}
		})
	}
}
