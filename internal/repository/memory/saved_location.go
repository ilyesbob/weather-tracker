package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"weathertracker/internal/domain"
)

// SavedLocationStore is an in-memory SavedLocationRepository. Entries are kept
// per user and ordered by DisplayOrder (User Story 1 comparison list).
type SavedLocationStore struct {
	mu     sync.RWMutex
	byUser map[string][]domain.SavedLocation
	now    func() time.Time
}

// NewSavedLocationStore creates an empty store.
func NewSavedLocationStore() *SavedLocationStore {
	return &SavedLocationStore{
		byUser: make(map[string][]domain.SavedLocation),
		now:    time.Now,
	}
}

// List returns the user's saved locations ordered by display order.
func (s *SavedLocationStore) List(_ context.Context, userID string) ([]domain.SavedLocation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.byUser[userID]
	out := make([]domain.SavedLocation, len(items))
	copy(out, items)
	return out, nil
}

// Add appends a location, rejecting duplicates and assigning the next order.
func (s *SavedLocationStore) Add(_ context.Context, userID string, sl domain.SavedLocation) (domain.SavedLocation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.byUser[userID] {
		if existing.LocationID == sl.LocationID {
			return domain.SavedLocation{}, fmt.Errorf("%w: location %d already saved", domain.ErrConflict, sl.LocationID)
		}
	}

	sl.DisplayOrder = len(s.byUser[userID])
	sl.SavedAt = s.now().UTC()
	s.byUser[userID] = append(s.byUser[userID], sl)
	return sl, nil
}

// Delete removes a location from the user's list and compacts the ordering.
func (s *SavedLocationStore) Delete(_ context.Context, userID string, locationID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := s.byUser[userID]
	for i, item := range items {
		if item.LocationID != locationID {
			continue
		}
		items = append(items[:i], items[i+1:]...)
		for j := range items {
			items[j].DisplayOrder = j
		}
		s.byUser[userID] = items
		return nil
	}
	return fmt.Errorf("%w: location %d for user %q", domain.ErrNotFound, locationID, userID)
}

// Count returns how many locations the user currently tracks.
func (s *SavedLocationStore) Count(_ context.Context, userID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.byUser[userID]), nil
}
