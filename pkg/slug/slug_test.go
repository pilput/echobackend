package slug

import "testing"

func TestMake(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		limit int
		want  string
	}{
		{"simple", "Guild Dokter Indonesia", 100, "guild-dokter-indonesia"},
		{"punctuation collapses", "Go & Rust  Devs!!", 100, "go-rust-devs"},
		{"trims edges", "  --Hello--  ", 100, "hello"},
		{"digits kept", "Angkatan 2026", 100, "angkatan-2026"},
		{"underscores are separators", "my_guild_name", 100, "my-guild-name"},
		{"truncates and re-trims", "abcdef ghij", 7, "abcdef"},
		{"non-latin preserved", "Комната", 100, "комната"},
		{"only punctuation yields empty", "!!!???", 100, ""},
		{"empty input", "", 100, ""},
		{"no limit", "a b c", 0, "a-b-c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Make(tt.in, tt.limit); got != tt.want {
				t.Errorf("Make(%q, %d) = %q, want %q", tt.in, tt.limit, got, tt.want)
			}
		})
	}
}
