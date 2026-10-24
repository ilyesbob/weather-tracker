package service

import (
	"context"
	"fmt"
	"strings"

	"weathertracker/internal/domain"
	"weathertracker/internal/repository"
)

// MaxSavedLocations caps the comparison list, matching User Story 1 ("до 5
// городов одновременно"). It is exported so handlers can reference the same
// constant in validation messages.
const MaxSavedLocations = 5

// LocationService implements the saved-location use cases backing User Story 1.
type LocationService struct {
	geocoder repository.Geocoder
	catalog  repository.LocationRepository
	saved    repository.SavedLocationRepository
}

// NewLocationService wires the location service dependencies.
func NewLocationService(geocoder repository.Geocoder, catalog repository.LocationRepository, saved repository.SavedLocationRepository) *LocationService {
	return &LocationService{geocoder: geocoder, catalog: catalog, saved: saved}
}

// Search resolves a query to locations and persists them in the catalogue so
// that every result carries a stable identifier the frontend can save.
func (s *LocationService) Search(ctx context.Context, query string, limit int) ([]domain.Location, error) {
	query = strings.TrimSpace(query)
	if len(query) < 2 {
		return nil, fmt.Errorf("%w: query must be at least 2 characters", domain.ErrValidation)
	}

	found, err := s.geocoder.Search(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search locations: %w", err)
	}

	out := make([]domain.Location, 0, len(found))
	for _, loc := range found {
		persisted, err := s.catalog.Upsert(ctx, loc)
		if err != nil {
			return nil, fmt.Errorf("persist location %q: %w", loc.Name, err)
		}
		out = append(out, persisted)
	}
	return out, nil
}

// AddSavedInput carries the data required to add a location to the comparison
// list. Either LocationID (a previously searched location) or coordinates must
// be provided.
type AddSavedInput struct {
	LocationID     int64    `json:"locationId"`
	Name           string   `json:"name"`
	Latitude       float64  `json:"latitude"`
	Longitude      float64  `json:"longitude"`
	CountryCode    string   `json:"countryCode"`
	Timezone       string   `json:"timezone"`
	Alias          string   `json:"alias"`
	ShoreNormalDeg *float64 `json:"shoreNormalDeg"`
}

// AddSaved adds a location to the user's comparison list, enforcing the cap of
// MaxSavedLocations.
func (s *LocationService) AddSaved(ctx context.Context, userID string, in AddSavedInput) (domain.SavedLocation, error) {
	count, err := s.saved.Count(ctx, userID)
	if err != nil {
		return domain.SavedLocation{}, fmt.Errorf("count saved locations: %w", err)
	}
	if count >= MaxSavedLocations {
		return domain.SavedLocation{}, fmt.Errorf("%w: comparison list is limited to %d cities", domain.ErrLimitExceeded, MaxSavedLocations)
	}

	loc, err := s.resolveLocation(ctx, in)
	if err != nil {
		return domain.SavedLocation{}, err
	}

	saved := domain.SavedLocation{
		LocationID:     loc.ID,
		Name:           loc.Name,
		Latitude:       loc.Latitude,
		Longitude:      loc.Longitude,
		CountryCode:    loc.CountryCode,
		Timezone:       loc.Timezone,
		Alias:          strings.TrimSpace(in.Alias),
		ShoreNormalDeg: in.ShoreNormalDeg,
	}
	created, err := s.saved.Add(ctx, userID, saved)
	if err != nil {
		return domain.SavedLocation{}, fmt.Errorf("add saved location: %w", err)
	}
	return created, nil
}

func (s *LocationService) resolveLocation(ctx context.Context, in AddSavedInput) (domain.Location, error) {
	if in.LocationID > 0 {
		loc, err := s.catalog.ByID(ctx, in.LocationID)
		if err != nil {
			return domain.Location{}, fmt.Errorf("resolve location %d: %w", in.LocationID, err)
		}
		return loc, nil
	}
	if strings.TrimSpace(in.Name) == "" {
		return domain.Location{}, fmt.Errorf("%w: name is required when locationId is absent", domain.ErrValidation)
	}
	return s.catalog.Upsert(ctx, domain.Location{
		Name:        strings.TrimSpace(in.Name),
		Latitude:    in.Latitude,
		Longitude:   in.Longitude,
		CountryCode: in.CountryCode,
		Timezone:    in.Timezone,
	})
}

// ListSaved returns the user's comparison list ordered by display order.
func (s *LocationService) ListSaved(ctx context.Context, userID string) ([]domain.SavedLocation, error) {
	items, err := s.saved.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list saved locations: %w", err)
	}
	return items, nil
}

// DeleteSaved removes a location from the user's comparison list.
func (s *LocationService) DeleteSaved(ctx context.Context, userID string, locationID int64) error {
	if err := s.saved.Delete(ctx, userID, locationID); err != nil {
		return fmt.Errorf("delete saved location: %w", err)
	}
	return nil
}
