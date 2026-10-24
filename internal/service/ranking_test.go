package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"weathertracker/internal/domain"
)

func tempCriteria() domain.RankingCriteria {
	return domain.RankingCriteria{
		Temperature:   domain.MetricPreference{Enabled: true, Weight: 1, Min: 18, Max: 26, Ideal: floatPtr(22)},
		Precipitation: domain.PrecipitationCriterion{Enabled: true, Weight: 1, MaxProbability: 20},
		Wind:          domain.MetricPreference{Enabled: true, Weight: 1, Min: 1, Max: 6},
	}
}

func TestRankingService_Rank_OrdersByScore(t *testing.T) {
	now := time.Now().UTC()
	// Berlin is cold, Madrid is comfortable: Madrid must lead.
	provider := providerFunc(func(_ context.Context, lat, _ float64, _ int) (domain.WeatherSnapshot, error) {
		current := domain.CurrentWeather{Temperature: 5, WindSpeed: 3, WeatherCode: 0, Condition: "Clear sky"}
		if lat < 45 {
			current = domain.CurrentWeather{Temperature: 23, WindSpeed: 3, WeatherCode: 0, Condition: "Clear sky"}
		}
		return snapshot(current, now), nil
	})

	svc := NewRankingService(nil, provider, 5*time.Minute)
	result, err := svc.Rank(context.Background(), RankingInput{
		Cities: []CityInput{
			{Name: "Berlin", Latitude: 52, Longitude: 13},
			{Name: "Madrid", Latitude: 40, Longitude: -3},
		},
		Criteria: tempCriteria(),
	})
	if err != nil {
		t.Fatalf("Rank() error: %v", err)
	}
	if result.Cities[0].Name != "Madrid" || !result.Cities[0].IsLeader || result.Cities[0].Rank != 1 {
		t.Fatalf("expected Madrid to lead, got %+v", result.Cities)
	}
	if result.Cities[1].Rank != 2 || result.Cities[1].IsLeader {
		t.Fatalf("expected Berlin ranked second without leader flag, got %+v", result.Cities[1])
	}
}

func TestRankingService_Rank_StaleFlag(t *testing.T) {
	old := time.Now().UTC().Add(-10 * time.Minute)
	provider := providerFunc(func(_ context.Context, _, _ float64, _ int) (domain.WeatherSnapshot, error) {
		return snapshot(domain.CurrentWeather{Temperature: 22}, old), nil
	})

	svc := NewRankingService(nil, provider, 5*time.Minute)
	result, err := svc.Rank(context.Background(), RankingInput{
		Cities:   []CityInput{{Name: "Paris", Latitude: 48, Longitude: 2}},
		Criteria: tempCriteria(),
	})
	if err != nil {
		t.Fatalf("Rank() error: %v", err)
	}
	if !result.Cities[0].IsStale {
		t.Fatal("expected stale flag for data older than the threshold")
	}
	if result.Cities[0].LastUpdated.IsZero() {
		t.Fatal("expected LastUpdated to be populated")
	}
}

func TestRankingService_Rank_UpstreamUnavailable(t *testing.T) {
	svc := NewRankingService(nil, errProvider{err: errBoom}, 5*time.Minute)
	result, err := svc.Rank(context.Background(), RankingInput{
		Cities:   []CityInput{{Name: "Paris", Latitude: 48, Longitude: 2}},
		Criteria: tempCriteria(),
	})
	if err != nil {
		t.Fatalf("Rank() should degrade gracefully, got error: %v", err)
	}
	if len(result.Cities) != 1 || len(result.Cities[0].Reasons) == 0 {
		t.Fatalf("expected the city with an explanatory reason, got %+v", result.Cities)
	}
}

func TestRankingService_Rank_ConsidersForecast(t *testing.T) {
	now := time.Now().UTC()
	criteria := domain.RankingCriteria{
		Temperature: domain.MetricPreference{Enabled: true, Weight: 1, Min: 18, Max: 26},
	}
	comfortable := domain.CurrentWeather{Temperature: 23}

	// Same current weather for both cities; only one has a cold forecast ahead.
	withoutForecast := NewRankingService(nil, providerFunc(func(context.Context, float64, float64, int) (domain.WeatherSnapshot, error) {
		return snapshot(comfortable, now), nil
	}), time.Minute)

	coldHours := hourlySeries(now.Add(time.Hour), rankingForecastHorizon, func(h domain.HourlyForecast) domain.HourlyForecast {
		h.Temperature = -5
		return h
	})
	withForecast := NewRankingService(nil, providerFunc(func(context.Context, float64, float64, int) (domain.WeatherSnapshot, error) {
		return snapshot(comfortable, now, coldHours...), nil
	}), time.Minute)

	rankOnce := func(svc *RankingService) float64 {
		t.Helper()
		res, err := svc.Rank(context.Background(), RankingInput{
			Cities:   []CityInput{{Name: "Paris", Latitude: 48, Longitude: 2}},
			Criteria: criteria,
		})
		if err != nil {
			t.Fatalf("Rank() error: %v", err)
		}
		return res.Cities[0].Score
	}

	if got, base := rankOnce(withForecast), rankOnce(withoutForecast); got >= base {
		t.Fatalf("expected forecast to lower the score: withForecast=%v withoutForecast=%v", got, base)
	}
}

func TestRankingService_Rank_Validation(t *testing.T) {
	svc := NewRankingService(nil, providerFunc(func(context.Context, float64, float64, int) (domain.WeatherSnapshot, error) {
		return domain.WeatherSnapshot{FetchedAt: time.Now()}, nil
	}), time.Minute)

	t.Run("no cities", func(t *testing.T) {
		if _, err := svc.Rank(context.Background(), RankingInput{}); !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("expected ErrValidation, got %v", err)
		}
	})

	t.Run("city without name", func(t *testing.T) {
		_, err := svc.Rank(context.Background(), RankingInput{Cities: []CityInput{{Latitude: 1, Longitude: 1}}})
		if !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("expected ErrValidation, got %v", err)
		}
	})

	t.Run("more than five cities", func(t *testing.T) {
		cities := make([]CityInput, MaxSavedLocations+1)
		for i := range cities {
			cities[i] = CityInput{Name: "C", Latitude: float64(i), Longitude: float64(i)}
		}
		if _, err := svc.Rank(context.Background(), RankingInput{Cities: cities}); !errors.Is(err, domain.ErrLimitExceeded) {
			t.Fatalf("expected ErrLimitExceeded, got %v", err)
		}
	})
}
