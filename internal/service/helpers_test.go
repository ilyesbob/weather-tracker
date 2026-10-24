package service

import (
	"context"
	"errors"
	"time"

	"weathertracker/internal/domain"
)

// providerFunc adapts a function to the WeatherProvider interface so tests can
// control the forecast without touching the network.
type providerFunc func(ctx context.Context, latitude, longitude float64, days int) (domain.WeatherSnapshot, error)

func (f providerFunc) Fetch(ctx context.Context, latitude, longitude float64, days int) (domain.WeatherSnapshot, error) {
	return f(ctx, latitude, longitude, days)
}

// errProvider always fails, to exercise graceful-degradation paths.
type errProvider struct{ err error }

func (p errProvider) Fetch(context.Context, float64, float64, int) (domain.WeatherSnapshot, error) {
	return domain.WeatherSnapshot{}, p.err
}

// geocoderFunc adapts a function to the Geocoder interface.
type geocoderFunc func(ctx context.Context, query string, limit int) ([]domain.Location, error)

func (f geocoderFunc) Search(ctx context.Context, query string, limit int) ([]domain.Location, error) {
	return f(ctx, query, limit)
}

// snapshot builds a fresh forecast snapshot for a coordinate.
func snapshot(current domain.CurrentWeather, fetchedAt time.Time, hourly ...domain.HourlyForecast) domain.WeatherSnapshot {
	return domain.WeatherSnapshot{
		Current:   current,
		Hourly:    hourly,
		FetchedAt: fetchedAt,
	}
}

// hourlySeries builds n consecutive hourly entries starting at start.
func hourlySeries(start time.Time, n int, mutate func(domain.HourlyForecast) domain.HourlyForecast) []domain.HourlyForecast {
	out := make([]domain.HourlyForecast, 0, n)
	for i := 0; i < n; i++ {
		h := domain.HourlyForecast{Time: start.Add(time.Duration(i) * time.Hour)}
		if mutate != nil {
			h = mutate(h)
		}
		out = append(out, h)
	}
	return out
}

var errBoom = errors.New("boom")

func floatPtr(f float64) *float64                     { return &f }
func levelPtr(l domain.SkillLevel) *domain.SkillLevel { return &l }
func int64Ptr(i int64) *int64                         { return &i }
