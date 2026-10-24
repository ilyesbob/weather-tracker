package memory

import (
	"context"
	"fmt"
	"sync"

	"weathertracker/internal/domain"
)

// WindsurfProfileStore is an in-memory WindsurfProfileRepository.
type WindsurfProfileStore struct {
	mu     sync.RWMutex
	byUser map[string]domain.WindsurfProfile
}

// NewWindsurfProfileStore creates an empty profile store.
func NewWindsurfProfileStore() *WindsurfProfileStore {
	return &WindsurfProfileStore{byUser: make(map[string]domain.WindsurfProfile)}
}

// Save upserts a rider profile.
func (s *WindsurfProfileStore) Save(_ context.Context, p domain.WindsurfProfile) (domain.WindsurfProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byUser[p.UserID] = p
	return p, nil
}

// ByUserID returns the stored profile or ErrNotFound.
func (s *WindsurfProfileStore) ByUserID(_ context.Context, userID string) (domain.WindsurfProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.byUser[userID]
	if !ok {
		return domain.WindsurfProfile{}, fmt.Errorf("%w: windsurf profile for user %q", domain.ErrNotFound, userID)
	}
	return p, nil
}
