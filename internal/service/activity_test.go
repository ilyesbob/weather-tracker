package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"weathertracker/internal/domain"
	"weathertracker/internal/repository/memory"
)

func TestActivityService_Recommend_Success(t *testing.T) {
	catalog := memory.NewLocationStore()
	saved := memory.NewSavedLocationStore()

	loc, err := catalog.Upsert(context.Background(), domain.Location{
		Name: "Moscow", Latitude: 55.75, Longitude: 37.61, CountryCode: "RU",
	})
	if err != nil {
		t.Fatalf("catalog upsert: %v", err)
	}

	baseTime := time.Date(2026, 6, 15, 8, 0, 0, 0, time.UTC)
	hourly := hourlySeries(baseTime, 12, func(h domain.HourlyForecast) domain.HourlyForecast {
		h.Temperature = 18.0
		h.PrecipitationProbability = 5.0
		h.WindSpeed = 3.5
		h.WeatherCode = 0
		return h
	})

	weather := providerFunc(func(_ context.Context, _, _ float64, _ int) (domain.WeatherSnapshot, error) {
		return snapshot(domain.CurrentWeather{Temperature: 18.0}, baseTime, hourly...), nil
	})

	svc := NewActivityService(catalog, saved, weather, 5*time.Minute)
	svc.now = func() time.Time { return baseTime }

	res, err := svc.Recommend(context.Background(), "user-1", ActivityInput{
		LocationID: &loc.ID,
		Date:       "2026-06-15",
		Activity:   "running",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Windows) == 0 {
		t.Fatalf("expected recommended windows, got none")
	}
	if !res.Windows[0].IsBest {
		t.Errorf("expected first window to be flagged as isBest")
	}
	if !res.Windows[0].MatchesConditions {
		t.Errorf("expected conditions to match")
	}
	if len(res.Windows[0].Reasons) < 3 {
		t.Errorf("expected at least 3 reasons, got %d", len(res.Windows[0].Reasons))
	}
}

func TestActivityService_Recommend_ClosestOptionWhenNoExactMatch(t *testing.T) {
	catalog := memory.NewLocationStore()
	saved := memory.NewSavedLocationStore()

	baseTime := time.Date(2026, 6, 15, 8, 0, 0, 0, time.UTC)
	hourly := hourlySeries(baseTime, 8, func(h domain.HourlyForecast) domain.HourlyForecast {
		h.Temperature = 5.0
		h.PrecipitationProbability = 90.0
		h.WindSpeed = 15.0
		h.WeatherCode = 61
		return h
	})

	weather := providerFunc(func(_ context.Context, _, _ float64, _ int) (domain.WeatherSnapshot, error) {
		return snapshot(domain.CurrentWeather{}, baseTime, hourly...), nil
	})

	svc := NewActivityService(catalog, saved, weather, 5*time.Minute)
	svc.now = func() time.Time { return baseTime }

	res, err := svc.Recommend(context.Background(), "user-1", ActivityInput{
		City: &CityInput{
			Name: "Saint Petersburg", Latitude: 59.93, Longitude: 30.33, CountryCode: "RU",
		},
		Date:     "2026-06-15",
		Activity: "running",
		Conditions: &domain.ActivityConditions{
			TempMin:                     15,
			TempMax:                     22,
			MaxPrecipitationProbability: 10,
			MaxWindSpeed:                5,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Message == "" {
		t.Errorf("expected message about no exact match, got empty")
	}
	if len(res.Windows) == 0 {
		t.Fatalf("expected closest window to be returned")
	}
	if !res.Windows[0].IsBest {
		t.Errorf("expected closest window to be marked isBest")
	}
	if res.Windows[0].MatchesConditions {
		t.Errorf("expected matchesConditions to be false")
	}
}

func TestActivityService_Recommend_StaleDataRejected(t *testing.T) {
	catalog := memory.NewLocationStore()
	saved := memory.NewSavedLocationStore()

	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	staleFetchedAt := now.Add(-1 * time.Hour)

	weather := providerFunc(func(_ context.Context, _, _ float64, _ int) (domain.WeatherSnapshot, error) {
		return snapshot(domain.CurrentWeather{}, staleFetchedAt, domain.HourlyForecast{Time: now}), nil
	})

	svc := NewActivityService(catalog, saved, weather, 5*time.Minute)
	svc.now = func() time.Time { return now }

	_, err := svc.Recommend(context.Background(), "user-1", ActivityInput{
		City:     &CityInput{Name: "Kazan", Latitude: 55.79, Longitude: 49.12},
		Date:     "2026-06-15",
		Activity: "cycling",
	})

	if err == nil {
		t.Fatalf("expected error on stale forecast data, got nil")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation for stale data, got %v", err)
	}
}

func TestActivityService_Recommend_PastDateRejected(t *testing.T) {
	catalog := memory.NewLocationStore()
	saved := memory.NewSavedLocationStore()

	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	weather := providerFunc(func(_ context.Context, _, _ float64, _ int) (domain.WeatherSnapshot, error) {
		return snapshot(domain.CurrentWeather{}, now), nil
	})

	svc := NewActivityService(catalog, saved, weather, 5*time.Minute)
	svc.now = func() time.Time { return now }

	_, err := svc.Recommend(context.Background(), "user-1", ActivityInput{
		City:     &CityInput{Name: "Sochi", Latitude: 43.58, Longitude: 39.72},
		Date:     "2026-06-10",
		Activity: "walking",
	})

	if err == nil {
		t.Fatalf("expected error for past date, got nil")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}
