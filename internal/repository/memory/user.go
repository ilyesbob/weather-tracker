// Package memory provides thread-safe, in-memory implementations of the
// repository interfaces. They make the MVP runnable without PostgreSQL/Redis
// while keeping the exact same contracts the production adapters will satisfy.
package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"weathertracker/internal/domain"
)

// UserStore is an in-memory UserRepository.
type UserStore struct {
	mu     sync.RWMutex
	byID   map[string]domain.User
	byMail map[string]string
}

// NewUserStore creates an empty user store.
func NewUserStore() *UserStore {
	return &UserStore{
		byID:   make(map[string]domain.User),
		byMail: make(map[string]string),
	}
}

// Create stores a new user, rejecting duplicate emails.
func (s *UserStore) Create(_ context.Context, u domain.User) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := normalizeEmail(u.Email)
	if _, ok := s.byMail[key]; ok {
		return domain.User{}, fmt.Errorf("%w: email %q", domain.ErrConflict, u.Email)
	}
	s.byID[u.ID] = u
	s.byMail[key] = u.ID
	return u, nil
}

// ByEmail returns a user by its email address.
func (s *UserStore) ByEmail(_ context.Context, email string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.byMail[normalizeEmail(email)]
	if !ok {
		return domain.User{}, fmt.Errorf("%w: email %q", domain.ErrNotFound, email)
	}
	return s.byID[id], nil
}

// ByID returns a user by its identifier.
func (s *UserStore) ByID(_ context.Context, id string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.byID[id]
	if !ok {
		return domain.User{}, fmt.Errorf("%w: user %q", domain.ErrNotFound, id)
	}
	return u, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
