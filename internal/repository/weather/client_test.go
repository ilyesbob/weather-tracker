package weather_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"weathertracker/internal/domain"
	"weathertracker/internal/repository/weather"
)

// newOpenMeteoStub serves canned forecast and geocoding payloads.
func newOpenMeteoStub(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/forecast", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"current": {"time":"2026-03-30T10:00","temperature_2m":14.2,"apparent_temperature":13.0,
				"relative_humidity_2m":65,"wind_speed_10m":4.1,"wind_direction_10m":200,
				"wind_gusts_10m":7.0,"weather_code":2},
			"hourly": {
				"time":["2026-03-30T10:00","2026-03-30T11:00"],
				"temperature_2m":[14.2,15.0],
				"apparent_temperature":[13.0,14.0],
				"relative_humidity_2m":[65,60],
				"precipitation_probability":[10,20],
				"wind_speed_10m":[4.1,5.0],
				"wind_direction_10m":[200,210],
				"wind_gusts_10m":[7.0,8.0],
				"weather_code":[2,3]
			}
		}`))
	})
	mux.HandleFunc("/v1/search", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[
			{"id":1,"name":"Berlin","latitude":52.52,"longitude":13.41,"country_code":"DE","timezone":"Europe/Berlin"}
		]}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestClient_Fetch(t *testing.T) {
	srv := newOpenMeteoStub(t)
	client := weather.NewClient(srv.URL, srv.URL, 2*time.Second)

	snap, err := client.Fetch(context.Background(), 52.52, 13.41, 1)
	if err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}
	if snap.Current.Temperature != 14.2 || snap.Current.Humidity != 65 {
		t.Fatalf("unexpected current weather: %+v", snap.Current)
	}
	if len(snap.Hourly) != 2 || snap.Hourly[1].WindSpeed != 5.0 {
		t.Fatalf("unexpected hourly forecast: %+v", snap.Hourly)
	}
	if snap.Current.Condition != "Partly cloudy" {
		t.Fatalf("expected weather code to map to a condition, got %q", snap.Current.Condition)
	}
	if snap.FetchedAt.IsZero() {
		t.Fatal("expected FetchedAt to be set")
	}
}

func TestClient_Search(t *testing.T) {
	srv := newOpenMeteoStub(t)
	client := weather.NewClient(srv.URL, srv.URL, 2*time.Second)

	locations, err := client.Search(context.Background(), "Berlin", 5)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(locations) != 1 || locations[0].CountryCode != "DE" {
		t.Fatalf("unexpected locations: %+v", locations)
	}
}

func TestClient_Fetch_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	client := weather.NewClient(srv.URL, srv.URL, 2*time.Second)
	if _, err := client.Fetch(context.Background(), 1, 1, 1); err == nil {
		t.Fatal("expected an error for a 500 response")
	}
}

// countingProvider records how many times the upstream is called.
type countingProvider struct {
	calls int
	snap  domain.WeatherSnapshot
	err   error
}

func (p *countingProvider) Fetch(context.Context, float64, float64, int) (domain.WeatherSnapshot, error) {
	p.calls++
	return p.snap, p.err
}

func TestCachedProvider_CachesAndFallsBack(t *testing.T) {
	upstream := &countingProvider{snap: domain.WeatherSnapshot{FetchedAt: time.Now().UTC()}}
	cached := weather.NewCachedProvider(upstream, time.Minute)
	ctx := context.Background()

	t.Run("second call is served from cache", func(t *testing.T) {
		if _, err := cached.Fetch(ctx, 52.52, 13.41, 1); err != nil {
			t.Fatalf("first Fetch() error: %v", err)
		}
		if _, err := cached.Fetch(ctx, 52.52, 13.41, 1); err != nil {
			t.Fatalf("second Fetch() error: %v", err)
		}
		if upstream.calls != 1 {
			t.Fatalf("upstream calls = %d, want 1", upstream.calls)
		}
	})

	t.Run("upstream failure serves the last snapshot", func(t *testing.T) {
		empty := weather.NewCachedProvider(&countingProvider{err: http.ErrHandlerTimeout}, time.Minute)
		if _, err := empty.Fetch(ctx, 1, 1, 1); err == nil {
			t.Fatal("expected an error when the cache is empty")
		}

		stable := &countingProvider{snap: domain.WeatherSnapshot{FetchedAt: time.Now().UTC()}}
		// A zero TTL makes every stored entry immediately expired, so the next
		// call reaches the (now failing) upstream and must fall back to cache.
		provider := weather.NewCachedProvider(stable, 0)
		first, err := provider.Fetch(ctx, 2, 2, 1)
		if err != nil {
			t.Fatalf("seed Fetch() error: %v", err)
		}
		stable.err = http.ErrHandlerTimeout
		second, err := provider.Fetch(ctx, 2, 2, 1)
		if err != nil {
			t.Fatalf("expected graceful fallback, got error: %v", err)
		}
		if !second.FetchedAt.Equal(first.FetchedAt) {
			t.Fatal("expected the last known snapshot to be returned")
		}
	})
}
