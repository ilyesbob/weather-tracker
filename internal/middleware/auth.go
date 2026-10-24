package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// TokenVerifier validates a raw access token and returns its subject.
type TokenVerifier interface {
	Verify(raw string) (string, error)
}

type contextKey string

const userIDKey contextKey = "userID"

// Auth enforces a valid `Authorization: Bearer <token>` header and stores the
// authenticated user ID in the request context.
func Auth(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := bearerToken(r)
			if !ok {
				writeUnauthorized(w, "missing bearer token")
				return
			}
			userID, err := verifier.Verify(raw)
			if err != nil {
				writeUnauthorized(w, "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserID extracts the authenticated user ID stored by Auth.
func UserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	return token, token != ""
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
