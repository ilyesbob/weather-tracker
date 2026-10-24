package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// Ensure no ambient variables leak into the test.
	for _, key := range []string{"APP_ADDR", "JWT_SECRET", "JWT_TTL", "WEATHER_CACHE_TTL", "STALE_THRESHOLD", "ALLOWED_ORIGINS"} {
		t.Setenv(key, "")
	}

	cfg := Load()
	if cfg.Addr != ":8080" {
		t.Fatalf("Addr = %q, want :8080", cfg.Addr)
	}
	if cfg.StaleThreshold != 5*time.Minute {
		t.Fatalf("StaleThreshold = %v, want 5m", cfg.StaleThreshold)
	}
	if len(cfg.AllowedOrigins) == 0 {
		t.Fatal("expected default allowed origins")
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("APP_ADDR", ":9090")
	t.Setenv("JWT_TTL", "2h")
	t.Setenv("STALE_THRESHOLD", "30s")
	t.Setenv("ALLOWED_ORIGINS", "http://a.test, http://b.test")

	cfg := Load()
	if cfg.Addr != ":9090" {
		t.Fatalf("Addr = %q, want :9090", cfg.Addr)
	}
	if cfg.JWTTTL != 2*time.Hour {
		t.Fatalf("JWTTTL = %v, want 2h", cfg.JWTTTL)
	}
	if cfg.StaleThreshold != 30*time.Second {
		t.Fatalf("StaleThreshold = %v, want 30s", cfg.StaleThreshold)
	}
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[1] != "http://b.test" {
		t.Fatalf("AllowedOrigins = %v", cfg.AllowedOrigins)
	}
}

func TestEnvDurationFallsBackOnGarbage(t *testing.T) {
	t.Setenv("JWT_TTL", "not-a-duration")
	if got := envDuration("JWT_TTL", time.Minute); got != time.Minute {
		t.Fatalf("envDuration() = %v, want fallback 1m", got)
	}
}
