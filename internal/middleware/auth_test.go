package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"weathertracker/internal/middleware"
)

type verifierFunc func(raw string) (string, error)

func (f verifierFunc) Verify(raw string) (string, error) { return f(raw) }

func TestAuthMiddleware(t *testing.T) {
	ok := func(string) (string, error) { return "user-42", nil }
	deny := func(string) (string, error) { return "", errors.New("bad token") }

	tests := []struct {
		name       string
		header     string
		verifier   middleware.TokenVerifier
		wantStatus int
		wantUser   string
	}{
		{name: "valid bearer token", header: "Bearer good", verifier: verifierFunc(ok), wantStatus: http.StatusOK, wantUser: "user-42"},
		{name: "missing header", header: "", verifier: verifierFunc(ok), wantStatus: http.StatusUnauthorized},
		{name: "wrong scheme", header: "Token good", verifier: verifierFunc(ok), wantStatus: http.StatusUnauthorized},
		{name: "rejected token", header: "Bearer bad", verifier: verifierFunc(deny), wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var seen string
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seen, _ = middleware.UserID(r.Context())
				w.WriteHeader(http.StatusOK)
			})
			handler := middleware.Auth(tt.verifier)(next)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantUser != "" && seen != tt.wantUser {
				t.Fatalf("user in context = %q, want %q", seen, tt.wantUser)
			}
		})
	}
}

func TestCORSAllowedOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := middleware.CORS([]string{"http://localhost:5173", "http://localhost:3000"})(next)

	t.Run("Vite origin is echoed", func(t *testing.T) {
		assertAllowedOrigin(t, handler, "http://localhost:5173", "http://localhost:5173")
	})

	t.Run("CRA origin is echoed", func(t *testing.T) {
		assertAllowedOrigin(t, handler, "http://localhost:3000", "http://localhost:3000")
	})

	t.Run("disallowed origin is not echoed", func(t *testing.T) {
		assertAllowedOrigin(t, handler, "http://evil.example.com", "")
	})

	t.Run("preflight returns 204", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("preflight status = %d, want 204", rec.Code)
		}
	})
}

func assertAllowedOrigin(t *testing.T, handler http.Handler, origin, want string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", origin)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != want {
		t.Fatalf("Allow-Origin for %q = %q, want %q", origin, got, want)
	}
}
