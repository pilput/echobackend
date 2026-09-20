package uid

import (
	"regexp"
	"testing"
	"time"
)

var canonical = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestNewV7_ShapeAndVersion(t *testing.T) {
	id, err := NewV7()
	if err != nil {
		t.Fatalf("NewV7 failed: %v", err)
	}
	// Postgres stores these in a uuid column, so anything outside the canonical
	// form (or the wrong version nibble) would be rejected on INSERT.
	if !canonical.MatchString(id) {
		t.Fatalf("expected a canonical version 7 UUID, got %q", id)
	}
}

func TestNewV7_Unique(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for range 1000 {
		id, err := NewV7()
		if err != nil {
			t.Fatalf("NewV7 failed: %v", err)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id generated: %s", id)
		}
		seen[id] = struct{}{}
	}
}

func TestNewV7_IsTimeOrdered(t *testing.T) {
	first, err := NewV7()
	if err != nil {
		t.Fatalf("NewV7 failed: %v", err)
	}
	// The timestamp lives in the leading 48 bits, so ids minted later must sort
	// after earlier ones — that is the whole reason for preferring v7 over v4.
	time.Sleep(2 * time.Millisecond)
	second, err := NewV7()
	if err != nil {
		t.Fatalf("NewV7 failed: %v", err)
	}
	if second <= first {
		t.Fatalf("expected %q to sort after %q", second, first)
	}
}
