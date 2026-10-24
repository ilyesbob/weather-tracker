// Package domain contains the core business entities of the Weather Tracker
// service. The types are intentionally free of any infrastructure concerns
// (HTTP, SQL, caching) so they can be reused by every layer.
package domain

import "errors"

// Sentinel errors returned by repositories and services. Handlers translate
// them into HTTP status codes, keeping transport concerns out of the business
// logic. Wrapping with fmt.Errorf("...: %w", err) preserves these identities.
var (
	// ErrNotFound signals that a requested entity does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict signals a uniqueness violation (e.g. duplicate email).
	ErrConflict = errors.New("already exists")
	// ErrUnauthorized signals missing or invalid credentials.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrLimitExceeded signals that a business limit was reached
	// (e.g. more than five cities in the comparison list).
	ErrLimitExceeded = errors.New("limit exceeded")
	// ErrValidation signals malformed or semantically invalid input.
	ErrValidation = errors.New("validation failed")
)
