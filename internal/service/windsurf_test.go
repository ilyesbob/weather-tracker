package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"weathertracker/internal/domain"
	"weathertracker/internal/repository/memory"
)

func windProvider(speed, direction, gusts float64, now time.Time) providerFunc {
	return func(context.Context, float64, float64, int) (domain.WeatherSnapshot, error) {
		hours := hourlySeries(now, 12, func(h domain.HourlyForecast) domain.HourlyForecast {
			h.WindSpeed = speed
			h.WindDirection = direction
			h.WindGusts = gusts
			return h
		})
		return domain.WeatherSnapshot{Hourly: hours, FetchedAt: now}, nil
	}
}

func TestWindsurfService_SaveProfile(t *testing.T) {
	tests := []struct {
		name    string
		input   ProfileInput
		wantErr error
	}{
		{name: "happy path", input: ProfileInput{Level: domain.LevelIntermediate, WindSpeedMin: 6, WindSpeedMax: 14}},
		{name: "invalid level", input: ProfileInput{Level: "wizard", WindSpeedMin: 6, WindSpeedMax: 14}, wantErr: domain.ErrValidation},
		{name: "inverted range", input: ProfileInput{Level: domain.LevelPro, WindSpeedMin: 14, WindSpeedMax: 6}, wantErr: domain.ErrValidation},
		{name: "negative wind", input: ProfileInput{Level: domain.LevelPro, WindSpeedMin: -1, WindSpeedMax: 6}, wantErr: domain.ErrValidation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewWindsurfService(memory.NewWindsurfProfileStore(), nil, nil, nil, time.Minute)
			_, err := svc.SaveProfile(context.Background(), "u1", tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("SaveProfile() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("SaveProfile() unexpected error: %v", err)
			}
		})
	}
}

func TestWindsurfService_Profile_NotFound(t *testing.T) {
	svc := NewWindsurfService(memory.NewWindsurfProfileStore(), nil, nil, nil, time.Minute)
	if _, err := svc.Profile(context.Background(), "ghost"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestWindsurfService_Recommend_OnshoreIsBest(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Hour)
	shore := 90.0
	lat, lon := 36.0, -5.6

	svc := NewWindsurfService(memory.NewWindsurfProfileStore(), nil, nil, windProvider(9, 90, 11, now), time.Minute)
	result, err := svc.Recommend(context.Background(), "u1", RecommendInput{
		Level:        levelPtr(domain.LevelIntermediate),
		WindSpeedMin: floatPtr(6),
		WindSpeedMax: floatPtr(14),
		Days:         1,
		SlotHours:    3,
		Spots:        []domain.Spot{{Name: "Tarifa", Latitude: &lat, Longitude: &lon, ShoreNormalDeg: &shore}},
	})
	if err != nil {
		t.Fatalf("Recommend() error: %v", err)
	}
	if len(result.Recommendations) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(result.Recommendations))
	}
	best := result.Recommendations[0]
	if !best.IsBest || best.Orientation != domain.OrientationOnshore {
		t.Fatalf("expected onshore best recommendation, got %+v", best)
	}
	if best.Score < 70 {
		t.Fatalf("expected a strong score, got %v", best.Score)
	}
	if len(best.Reasons) == 0 {
		t.Fatal("expected explanatory reasons")
	}
}

func TestWindsurfService_Recommend_OffshorePenalisedForBeginners(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Hour)
	shore := 90.0
	lat, lon := 36.0, -5.6
	// Wind from the land (270°) is offshore for a shore facing 90°.
	provider := windProvider(9, 270, 11, now)

	recommend := func(level domain.SkillLevel) float64 {
		svc := NewWindsurfService(memory.NewWindsurfProfileStore(), nil, nil, provider, time.Minute)
		res, err := svc.Recommend(context.Background(), "u1", RecommendInput{
			Level:        levelPtr(level),
			WindSpeedMin: floatPtr(6),
			WindSpeedMax: floatPtr(14),
			Days:         1,
			SlotHours:    3,
			Spots:        []domain.Spot{{Name: "Spot", Latitude: &lat, Longitude: &lon, ShoreNormalDeg: &shore}},
		})
		if err != nil {
			t.Fatalf("Recommend() error: %v", err)
		}
		if res.Recommendations[0].Orientation != domain.OrientationOffshore {
			t.Fatalf("expected offshore orientation, got %s", res.Recommendations[0].Orientation)
		}
		return res.Recommendations[0].Score
	}

	if beginner, pro := recommend(domain.LevelBeginner), recommend(domain.LevelPro); beginner >= pro {
		t.Fatalf("expected beginners to score offshore lower: beginner=%v pro=%v", beginner, pro)
	}
}

func TestWindsurfService_Recommend_FallsBackToSavedLocations(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Hour)
	saved := memory.NewSavedLocationStore()
	if _, err := saved.Add(context.Background(), "u1", domain.SavedLocation{
		LocationID: 1, Name: "Berlin", Latitude: 52.5, Longitude: 13.4,
	}); err != nil {
		t.Fatalf("seed saved location: %v", err)
	}

	svc := NewWindsurfService(memory.NewWindsurfProfileStore(), saved, nil, windProvider(9, 90, 11, now), time.Minute)
	result, err := svc.Recommend(context.Background(), "u1", RecommendInput{Days: 1, SlotHours: 3})
	if err != nil {
		t.Fatalf("Recommend() error: %v", err)
	}
	if len(result.Recommendations) != 1 || result.Recommendations[0].SpotName != "Berlin" {
		t.Fatalf("expected saved location to be used, got %+v", result.Recommendations)
	}
}

func TestWindsurfService_Recommend_RequiresSpots(t *testing.T) {
	svc := NewWindsurfService(memory.NewWindsurfProfileStore(), memory.NewSavedLocationStore(), nil, windProvider(9, 90, 11, time.Now()), time.Minute)
	if _, err := svc.Recommend(context.Background(), "u1", RecommendInput{}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation when no spots are available, got %v", err)
	}
}

func TestWindsurfService_Recommend_NoData(t *testing.T) {
	svc := NewWindsurfService(memory.NewWindsurfProfileStore(), nil, nil, errProvider{err: errBoom}, time.Minute)
	lat, lon := 36.0, -5.6
	result, err := svc.Recommend(context.Background(), "u1", RecommendInput{
		Spots: []domain.Spot{{Name: "Spot", Latitude: &lat, Longitude: &lon}},
	})
	if err != nil {
		t.Fatalf("Recommend() should not fail when data is unavailable: %v", err)
	}
	if len(result.Recommendations) != 0 || result.Message == "" {
		t.Fatalf("expected empty recommendations with a message, got %+v", result)
	}
}

func TestWindsurfService_Recommend_ResolvesLocationID(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Hour)
	catalog := memory.NewLocationStore()
	loc, err := catalog.Upsert(context.Background(), domain.Location{Name: "Cadiz", Latitude: 36.5, Longitude: -6.3})
	if err != nil {
		t.Fatalf("seed catalog: %v", err)
	}

	svc := NewWindsurfService(memory.NewWindsurfProfileStore(), nil, catalog, windProvider(9, 90, 11, now), time.Minute)
	result, err := svc.Recommend(context.Background(), "u1", RecommendInput{
		Days: 1, SlotHours: 3,
		Spots: []domain.Spot{{LocationID: int64Ptr(loc.ID)}},
	})
	if err != nil {
		t.Fatalf("Recommend() error: %v", err)
	}
	if len(result.Recommendations) != 1 || result.Recommendations[0].SpotName != "Cadiz" {
		t.Fatalf("expected spot resolved from catalog, got %+v", result.Recommendations)
	}
}
