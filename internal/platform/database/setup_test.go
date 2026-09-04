package database

import (
	"testing"
	"time"
)

// Zero means "not configured" for every pool knob, because config.envInt /
// envDuration hand back the zero value when the variable is absent. A regression
// here would silently apply database/sql's own defaults (unlimited open
// connections, 2 idle) instead of the intended ones.
func TestDefaultInt(t *testing.T) {
	if got := defaultInt(0, 25); got != 25 {
		t.Errorf("defaultInt(0, 25) = %d, want 25", got)
	}
	if got := defaultInt(10, 25); got != 10 {
		t.Errorf("defaultInt(10, 25) = %d, want 10", got)
	}
	if got := defaultInt(-1, 25); got != -1 {
		t.Errorf("defaultInt(-1, 25) = %d, want -1 (negative means unlimited, not unset)", got)
	}
}

func TestDefaultDuration(t *testing.T) {
	if got := defaultDuration(0, 5*time.Minute); got != 5*time.Minute {
		t.Errorf("defaultDuration(0, 5m) = %v, want 5m", got)
	}
	if got := defaultDuration(time.Second, 5*time.Minute); got != time.Second {
		t.Errorf("defaultDuration(1s, 5m) = %v, want 1s", got)
	}
}
