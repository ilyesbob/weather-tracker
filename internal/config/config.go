// Package config centralises runtime configuration. Values are read from the
// environment with sensible development defaults so the service starts with a
// single `go run` command.
package config

import (
	"os"
	"strings"
	"time"
)

// Config holds every tunable of the API process.
type Config struct {
	// Addr is the TCP address the HTTP server listens on.
	Addr string
	// JWTSecret signs and verifies access tokens.
	JWTSecret string
	// JWTTTL is the access token lifetime.
	JWTTTL time.Duration
	// WeatherBaseURL is the Open-Meteo forecast endpoint.
	WeatherBaseURL string
	// GeocodingBaseURL is the Open-Meteo geocoding endpoint.
	GeocodingBaseURL string
	// WeatherCacheTTL controls how long a weather snapshot is considered fresh.
	WeatherCacheTTL time.Duration
	// StaleThreshold is the age beyond which data is flagged as stale in
	// responses (User Story 1 requirement: 5 minutes).
	StaleThreshold time.Duration
	// UpstreamTimeout bounds every outbound HTTP call to Open-Meteo.
	UpstreamTimeout time.Duration
	// AllowedOrigins lists the browser origins permitted by CORS. It targets
	// the Vite (5173) and CRA (3000) React dev servers by default.
	AllowedOrigins []string
}

// Load builds a Config from environment variables, falling back to defaults.
func Load() Config {
	return Config{
		Addr:             env("APP_ADDR", ":8080"),
		JWTSecret:        env("JWT_SECRET", "dev-secret-change-me"),
		JWTTTL:           envDuration("JWT_TTL", 24*time.Hour),
		WeatherBaseURL:   env("WEATHER_BASE_URL", "https://api.open-meteo.com"),
		GeocodingBaseURL: env("GEOCODING_BASE_URL", "https://geocoding-api.open-meteo.com"),
		WeatherCacheTTL:  envDuration("WEATHER_CACHE_TTL", 5*time.Minute),
		StaleThreshold:   envDuration("STALE_THRESHOLD", 5*time.Minute),
		UpstreamTimeout:  envDuration("UPSTREAM_TIMEOUT", 5*time.Second),
		AllowedOrigins:   envList("ALLOWED_ORIGINS", defaultOrigins),
	}
}

var defaultOrigins = []string{
	"http://localhost:3000",
	"http://127.0.0.1:3000",
	"http://localhost:5173",
	"http://127.0.0.1:5173",
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func envList(key string, fallback []string) []string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
