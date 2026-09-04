package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnvString(t *testing.T) {
	const primary = "ECHOBACKEND_TEST_ENV_STRING_PRIMARY"
	const fallback = "ECHOBACKEND_TEST_ENV_STRING_FALLBACK"

	if got := envString([]string{primary, fallback}, "default"); got != "default" {
		t.Fatalf("expected default, got %q", got)
	}

	t.Setenv(fallback, "fallback-value")
	if got := envString([]string{primary, fallback}, "default"); got != "fallback-value" {
		t.Fatalf("expected fallback value, got %q", got)
	}

	t.Setenv(primary, "primary-value")
	if got := envString([]string{primary, fallback}, "default"); got != "primary-value" {
		t.Fatalf("expected primary value, got %q", got)
	}
}

func TestEnvInt(t *testing.T) {
	const primary = "ECHOBACKEND_TEST_ENV_INT_PRIMARY"
	const fallback = "ECHOBACKEND_TEST_ENV_INT_FALLBACK"

	if got := envInt([]string{primary, fallback}, 10); got != 10 {
		t.Fatalf("expected default, got %d", got)
	}

	t.Setenv(fallback, "20")
	if got := envInt([]string{primary, fallback}, 10); got != 20 {
		t.Fatalf("expected fallback int, got %d", got)
	}

	t.Setenv(primary, "invalid")
	if got := envInt([]string{primary, fallback}, 10); got != 10 {
		t.Fatalf("expected default for invalid primary, got %d", got)
	}
}

func TestEnvBool(t *testing.T) {
	const primary = "ECHOBACKEND_TEST_ENV_BOOL_PRIMARY"
	const fallback = "ECHOBACKEND_TEST_ENV_BOOL_FALLBACK"

	if got := envBool([]string{primary, fallback}, true); got != true {
		t.Fatalf("expected default true, got %v", got)
	}

	t.Setenv(fallback, "false")
	if got := envBool([]string{primary, fallback}, true); got != false {
		t.Fatalf("expected fallback false, got %v", got)
	}

	t.Setenv(primary, "not-a-bool")
	if got := envBool([]string{primary, fallback}, true); got != true {
		t.Fatalf("expected default for invalid primary, got %v", got)
	}
}

func TestEnvDuration(t *testing.T) {
	const primary = "ECHOBACKEND_TEST_ENV_DURATION_PRIMARY"
	const fallback = "ECHOBACKEND_TEST_ENV_DURATION_FALLBACK"
	defaultValue := 5 * time.Second

	if got := envDuration([]string{primary, fallback}, defaultValue); got != defaultValue {
		t.Fatalf("expected default duration, got %s", got)
	}

	t.Setenv(fallback, "2m")
	if got := envDuration([]string{primary, fallback}, defaultValue); got != 2*time.Minute {
		t.Fatalf("expected fallback duration, got %s", got)
	}

	t.Setenv(primary, "invalid")
	if got := envDuration([]string{primary, fallback}, defaultValue); got != defaultValue {
		t.Fatalf("expected default for invalid primary, got %s", got)
	}
}

func TestLoadDotEnv(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	const keyNormal = "TEST_DOTENV_KEY_NORMAL"
	const keyQuoted = "TEST_DOTENV_KEY_QUOTED"
	const keySingleQuoted = "TEST_DOTENV_KEY_SINGLE"
	const keyExisting = "TEST_DOTENV_KEY_EXISTING"

	t.Setenv(keyExisting, "original_value")

	content := `
# Comment line
TEST_DOTENV_KEY_NORMAL=normal_val
TEST_DOTENV_KEY_QUOTED="quoted val"
TEST_DOTENV_KEY_SINGLE='single val'
TEST_DOTENV_KEY_EXISTING=new_val
INVALID_LINE_WITHOUT_EQUALS
`
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test .env: %v", err)
	}

	if err := loadDotEnv(envPath); err != nil {
		t.Fatalf("failed to load .env: %v", err)
	}

	if got := os.Getenv(keyNormal); got != "normal_val" {
		t.Errorf("expected normal_val, got %q", got)
	}
	if got := os.Getenv(keyQuoted); got != "quoted val" {
		t.Errorf("expected 'quoted val', got %q", got)
	}
	if got := os.Getenv(keySingleQuoted); got != "single val" {
		t.Errorf("expected 'single val', got %q", got)
	}
	if got := os.Getenv(keyExisting); got != "original_value" {
		t.Errorf("expected original_value (not overwritten), got %q", got)
	}

	// Missing file returns error without panic
	if err := loadDotEnv(filepath.Join(tmpDir, "does-not-exist.env")); err == nil {
		t.Error("expected error for missing file")
	}
}
