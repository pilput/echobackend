package config

import (
	"os"
	"strconv"
	"time"
)

// envString returns the first set environment variable from keys, or defaultValue.
func envString(keys []string, defaultValue string) string {
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			return v
		}
	}
	return defaultValue
}

// envInt returns the first successfully parsed int from set env keys, or defaultValue.
// If a key is set but fails to parse, defaultValue is returned (subsequent keys are ignored).
func envInt(keys []string, defaultValue int) int {
	for _, k := range keys {
		if s, ok := os.LookupEnv(k); ok {
			if n, err := strconv.Atoi(s); err == nil {
				return n
			}
			return defaultValue
		}
	}
	return defaultValue
}

// envBool returns the first successfully parsed bool from set env keys, or defaultValue.
// If a key is set but fails to parse, defaultValue is returned (subsequent keys are ignored).
func envBool(keys []string, defaultValue bool) bool {
	for _, k := range keys {
		if s, ok := os.LookupEnv(k); ok {
			if b, err := strconv.ParseBool(s); err == nil {
				return b
			}
			return defaultValue
		}
	}
	return defaultValue
}

// envDuration returns the first successfully parsed Go duration (e.g. "3m", "1h30m")
// from set env keys, or defaultValue. If a key is set but fails to parse,
// defaultValue is returned (subsequent keys are ignored).
func envDuration(keys []string, defaultValue time.Duration) time.Duration {
	for _, k := range keys {
		if s, ok := os.LookupEnv(k); ok {
			if d, err := time.ParseDuration(s); err == nil {
				return d
			}
			return defaultValue
		}
	}
	return defaultValue
}

// resolveJWTExpiry returns JWT_EXPIRY (a Go duration string, e.g. "15m"), or
// falls back to the legacy JWT_EXPIRY_HOURS (whole hours) for backward
// compatibility with existing deployments, or defaultValue if neither is set.
func resolveJWTExpiry(defaultValue time.Duration) time.Duration {
	if s, ok := os.LookupEnv("JWT_EXPIRY"); ok {
		if d, err := time.ParseDuration(s); err == nil {
			return d
		}
	}
	if s, ok := os.LookupEnv("JWT_EXPIRY_HOURS"); ok {
		if h, err := strconv.Atoi(s); err == nil {
			return time.Duration(h) * time.Hour
		}
	}
	return defaultValue
}
