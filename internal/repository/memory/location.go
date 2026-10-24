package memory

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"weathertracker/internal/domain"
)

// LocationStore is an in-memory LocationRepository keyed by coordinates.
type LocationStore struct {
	mu      sync.RWMutex
	byID    map[int64]domain.Location
	byCoord map[string]int64
	nextID  int64
	now     func() time.Time
}

// NewLocationStore creates an empty location catalogue.
func NewLocationStore() *LocationStore {
	return &LocationStore{
		byID:    make(map[int64]domain.Location),
		byCoord: make(map[string]int64),
		now:     time.Now,
	}
}

// Upsert inserts the location or returns the previously stored point with the
// same rounded coordinates. Coordinates are rounded to four decimal places
// (~11 m) to tolerate floating point noise from the geocoder.
func (s *LocationStore) Upsert(_ context.Context, l domain.Location) (domain.Location, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := coordKey(l.Latitude, l.Longitude)
	if id, ok := s.byCoord[key]; ok {
		return s.byID[id], nil
	}

	s.nextID++
	l.ID = s.nextID
	if l.CreatedAt.IsZero() {
		l.CreatedAt = s.now().UTC()
	}
	s.byID[l.ID] = l
	s.byCoord[key] = l.ID
	return l, nil
}

// ByID returns a location by its identifier.
func (s *LocationStore) ByID(_ context.Context, id int64) (domain.Location, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	l, ok := s.byID[id]
	if !ok {
		return domain.Location{}, fmt.Errorf("%w: location %d", domain.ErrNotFound, id)
	}
	return l, nil
}

func coordKey(lat, lon float64) string {
	return fmt.Sprintf("%.4f:%.4f", math.Round(lat*1e4)/1e4, math.Round(lon*1e4)/1e4)
}
