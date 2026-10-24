package service

import (
	"context"
	"errors"
	"testing"

	"weathertracker/internal/domain"
	"weathertracker/internal/repository/memory"
)

func newLocationService(geocoder geocoderFunc) (*LocationService, *memory.LocationStore, *memory.SavedLocationStore) {
	catalog := memory.NewLocationStore()
	saved := memory.NewSavedLocationStore()
	return NewLocationService(geocoder, catalog, saved), catalog, saved
}

func TestLocationService_Search(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		query   string
		geocode geocoderFunc
		wantLen int
		wantErr error
	}{
		{
			name:  "happy path persists and assigns IDs",
			query: "Berlin",
			geocode: func(context.Context, string, int) ([]domain.Location, error) {
				return []domain.Location{{Name: "Berlin", Latitude: 52.52, Longitude: 13.41, CountryCode: "DE"}}, nil
			},
			wantLen: 1,
		},
		{
			name:    "too short query",
			query:   "B",
			geocode: func(context.Context, string, int) ([]domain.Location, error) { return nil, nil },
			wantErr: domain.ErrValidation,
		},
		{
			name:  "geocoder failure is wrapped",
			query: "Berlin",
			geocode: func(context.Context, string, int) ([]domain.Location, error) {
				return nil, errBoom
			},
			wantErr: errBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _ := newLocationService(tt.geocode)
			got, err := svc.Search(ctx, tt.query, 10)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Search() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Search() unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("Search() len = %d, want %d", len(got), tt.wantLen)
			}
			if got[0].ID == 0 {
				t.Fatal("expected persisted location to have a non-zero ID")
			}
		})
	}
}

func TestLocationService_AddSaved(t *testing.T) {
	ctx := context.Background()
	geocoder := func(context.Context, string, int) ([]domain.Location, error) { return nil, nil }

	t.Run("adds inline location", func(t *testing.T) {
		svc, _, _ := newLocationService(geocoder)
		saved, err := svc.AddSaved(ctx, "u1", AddSavedInput{Name: "Paris", Latitude: 48.85, Longitude: 2.35, CountryCode: "FR"})
		if err != nil {
			t.Fatalf("AddSaved() error: %v", err)
		}
		if saved.Name != "Paris" || saved.DisplayOrder != 0 {
			t.Fatalf("unexpected saved location: %+v", saved)
		}
	})

	t.Run("missing name and id is invalid", func(t *testing.T) {
		svc, _, _ := newLocationService(geocoder)
		if _, err := svc.AddSaved(ctx, "u1", AddSavedInput{}); !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("expected ErrValidation, got %v", err)
		}
	})

	t.Run("resolves by location ID", func(t *testing.T) {
		svc, catalog, _ := newLocationService(geocoder)
		loc, err := catalog.Upsert(ctx, domain.Location{Name: "Madrid", Latitude: 40.42, Longitude: -3.7})
		if err != nil {
			t.Fatalf("seed location: %v", err)
		}
		saved, err := svc.AddSaved(ctx, "u1", AddSavedInput{LocationID: loc.ID})
		if err != nil {
			t.Fatalf("AddSaved() error: %v", err)
		}
		if saved.LocationID != loc.ID || saved.Name != "Madrid" {
			t.Fatalf("unexpected saved location: %+v", saved)
		}
	})

	t.Run("unknown location ID", func(t *testing.T) {
		svc, _, _ := newLocationService(geocoder)
		if _, err := svc.AddSaved(ctx, "u1", AddSavedInput{LocationID: 999}); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("duplicate is rejected", func(t *testing.T) {
		svc, _, _ := newLocationService(geocoder)
		in := AddSavedInput{Name: "Paris", Latitude: 48.85, Longitude: 2.35}
		if _, err := svc.AddSaved(ctx, "u1", in); err != nil {
			t.Fatalf("first add failed: %v", err)
		}
		if _, err := svc.AddSaved(ctx, "u1", in); !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("expected ErrConflict, got %v", err)
		}
	})

	t.Run("enforces the five city limit", func(t *testing.T) {
		svc, _, _ := newLocationService(geocoder)
		for i := 0; i < MaxSavedLocations; i++ {
			in := AddSavedInput{Name: "City", Latitude: float64(i), Longitude: float64(i)}
			if _, err := svc.AddSaved(ctx, "u1", in); err != nil {
				t.Fatalf("add %d failed: %v", i, err)
			}
		}
		_, err := svc.AddSaved(ctx, "u1", AddSavedInput{Name: "Sixth", Latitude: 50, Longitude: 50})
		if !errors.Is(err, domain.ErrLimitExceeded) {
			t.Fatalf("expected ErrLimitExceeded, got %v", err)
		}
	})
}

func TestLocationService_ListSavedAndDelete(t *testing.T) {
	ctx := context.Background()
	geocoder := func(context.Context, string, int) ([]domain.Location, error) { return nil, nil }
	svc, _, _ := newLocationService(geocoder)

	a, err := svc.AddSaved(ctx, "u1", AddSavedInput{Name: "A", Latitude: 1, Longitude: 1})
	if err != nil {
		t.Fatalf("add A: %v", err)
	}
	if _, err := svc.AddSaved(ctx, "u1", AddSavedInput{Name: "B", Latitude: 2, Longitude: 2}); err != nil {
		t.Fatalf("add B: %v", err)
	}

	items, err := svc.ListSaved(ctx, "u1")
	if err != nil {
		t.Fatalf("ListSaved() error: %v", err)
	}
	if len(items) != 2 || items[1].DisplayOrder != 1 {
		t.Fatalf("unexpected list: %+v", items)
	}

	if err := svc.DeleteSaved(ctx, "u1", a.LocationID); err != nil {
		t.Fatalf("DeleteSaved() error: %v", err)
	}
	items, _ = svc.ListSaved(ctx, "u1")
	if len(items) != 1 || items[0].Name != "B" || items[0].DisplayOrder != 0 {
		t.Fatalf("expected ordering to be compacted, got %+v", items)
	}

	if err := svc.DeleteSaved(ctx, "u1", 999); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
