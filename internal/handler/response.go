// Package handler implements the HTTP transport layer. Each handler is thin:
// it decodes the request, calls a service method and encodes the result. All
// handlers carry swaggo annotations so the OpenAPI document stays in sync with
// the code.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"weathertracker/internal/domain"
)

// maxRequestBodyBytes caps request bodies to protect the server from oversized
// payloads. All current endpoints use small JSON objects, so 1 MiB is generous.
const maxRequestBodyBytes = 1 << 20

// errorResponse is the uniform error envelope consumed by the React client.
// Every 4xx/5xx response from this service has exactly this shape, which lets
// the frontend handle failures with a single code path.
type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// The status line is already written, so we can only log here.
		slog.Error("encode response", slog.String("error", err.Error()))
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

// handleError maps a domain error to the appropriate HTTP status while keeping
// the client-facing message stable. Unknown errors are hidden behind a generic
// 500 message but logged with full detail for operators.
func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrLimitExceeded):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	default:
		slog.Error("unhandled error", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

// decodeJSON reads and validates the request body. It bounds the body size,
// rejects unknown fields (surfacing frontend typos early) and requires exactly
// one JSON object.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errors.Join(domain.ErrValidation, err)
	}
	if dec.More() {
		return fmt.Errorf("%w: request body must contain a single JSON object", domain.ErrValidation)
	}
	return nil
}
