// Package repository declares the persistence and external-provider contracts
// used by the service layer. Following the Go proverb "accept interfaces,
// return structs", the service layer depends on these small, single-purpose
// interfaces while the concrete implementations live in sub-packages
// (repository/memory, repository/weather).
package repository

import (
	"context"

	"weathertracker/internal/domain"
)

// UserRepository stores user accounts.
type UserRepository interface {
	Create(ctx context.Context, u domain.User) (domain.User, error)
	ByEmail(ctx context.Context, email string) (domain.User, error)
	ByID(ctx context.Context, id string) (domain.User, error)
}

// LocationRepository stores the shared catalogue of geocoded points.
type LocationRepository interface {
	// Upsert inserts a location or returns the existing one matched by
	// coordinates, guaranteeing a stable identifier for the same point.
	Upsert(ctx context.Context, l domain.Location) (domain.Location, error)
	ByID(ctx context.Context, id int64) (domain.Location, error)
}

// SavedLocationRepository stores the per-user comparison list (User Story 1).
type SavedLocationRepository interface {
	List(ctx context.Context, userID string) ([]domain.SavedLocation, error)
	Add(ctx context.Context, userID string, sl domain.SavedLocation) (domain.SavedLocation, error)
	Delete(ctx context.Context, userID string, locationID int64) error
	Count(ctx context.Context, userID string) (int, error)
}

// WindsurfProfileRepository stores rider profiles (User Story 2).
type WindsurfProfileRepository interface {
	Save(ctx context.Context, p domain.WindsurfProfile) (domain.WindsurfProfile, error)
	ByUserID(ctx context.Context, userID string) (domain.WindsurfProfile, error)
}

// Geocoder resolves a free-text query into candidate locations.
type Geocoder interface {
	Search(ctx context.Context, query string, limit int) ([]domain.Location, error)
}

// WeatherProvider returns current weather and hourly forecast for a coordinate.
type WeatherProvider interface {
	Fetch(ctx context.Context, latitude, longitude float64, forecastDays int) (domain.WeatherSnapshot, error)
}
