package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"weathertracker/internal/domain"
	"weathertracker/internal/handler"
	"weathertracker/internal/repository/memory"
	"weathertracker/internal/service"
)

// providerFn lets each test shape the forecast returned by the fake provider.
type providerFn func(ctx context.Context, latitude, longitude float64, days int) (domain.WeatherSnapshot, error)

func (f providerFn) Fetch(ctx context.Context, lat, lon float64, days int) (domain.WeatherSnapshot, error) {
	return f(ctx, lat, lon, days)
}

// geocoderFn is a scripted Geocoder used by the location search handler.
type geocoderFn func(ctx context.Context, query string, limit int) ([]domain.Location, error)

func (f geocoderFn) Search(ctx context.Context, query string, limit int) ([]domain.Location, error) {
	return f(ctx, query, limit)
}

type testAPI struct {
	router http.Handler
	auth   *service.AuthService
}

// newTestAPI wires the real router with in-memory adapters and deterministic
// fakes, so handler behaviour (status codes, JSON, auth, CORS) is exercised
// without any external dependency.
func newTestAPI(t *testing.T) testAPI {
	t.Helper()

	users := memory.NewUserStore()
	catalog := memory.NewLocationStore()
	saved := memory.NewSavedLocationStore()
	profiles := memory.NewWindsurfProfileStore()

	auth := service.NewAuthService(users, "test-secret", time.Hour)

	geocoder := geocoderFn(func(_ context.Context, _ string, _ int) ([]domain.Location, error) {
		return []domain.Location{{Name: "Berlin", Latitude: 52.52, Longitude: 13.41, CountryCode: "DE"}}, nil
	})

	provider := providerFn(func(_ context.Context, lat, _ float64, _ int) (domain.WeatherSnapshot, error) {
		current := domain.CurrentWeather{Temperature: 5, WindSpeed: 3, WeatherCode: 0, Condition: "Clear sky"}
		if lat < 45 {
			current = domain.CurrentWeather{Temperature: 23, WindSpeed: 3, WeatherCode: 0, Condition: "Clear sky"}
		}
		hours := make([]domain.HourlyForecast, 0, 12)
		now := time.Now().UTC().Truncate(time.Hour)
		for i := 0; i < 12; i++ {
			hours = append(hours, domain.HourlyForecast{Time: now.Add(time.Duration(i) * time.Hour), WindSpeed: 9, WindDirection: 90, WindGusts: 11})
		}
		return domain.WeatherSnapshot{Current: current, Hourly: hours, FetchedAt: time.Now().UTC()}, nil
	})

	locSvc := service.NewLocationService(geocoder, catalog, saved)
	rankSvc := service.NewRankingService(catalog, provider, 5*time.Minute)
	wsSvc := service.NewWindsurfService(profiles, saved, catalog, provider, 5*time.Minute)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := handler.NewRouter(handler.Deps{
		Auth:           auth,
		Locations:      locSvc,
		Ranking:        rankSvc,
		Windsurf:       wsSvc,
		AllowedOrigins: []string{"http://localhost:5173"},
		Logger:         logger,
	})
	return testAPI{router: router, auth: auth}
}

// tokenFor registers a user and returns a valid access token.
func (api testAPI) tokenFor(t *testing.T) string {
	t.Helper()
	_, token, err := api.auth.Register(context.Background(), "rider@example.com", "s3cret-passphrase")
	if err != nil {
		t.Fatalf("seed token: %v", err)
	}
	return token
}

func (api testAPI) do(t *testing.T, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	api.router.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return out
}

func TestHealthz(t *testing.T) {
	api := newTestAPI(t)
	rec := api.do(t, http.MethodGet, "/healthz", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestAuthHandlers(t *testing.T) {
	credentials := map[string]string{"email": "rider@example.com", "password": "s3cret-passphrase"}

	t.Run("register happy path", func(t *testing.T) {
		api := newTestAPI(t)
		rec := api.do(t, http.MethodPost, "/api/v1/auth/register", "", credentials)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", rec.Code, rec.Body.String())
		}
		resp := decode[struct {
			Token string `json:"token"`
			User  struct {
				ID string `json:"id"`
			} `json:"user"`
		}](t, rec)
		if resp.Token == "" || resp.User.ID == "" {
			t.Fatalf("expected token and user ID, got %+v", resp)
		}
	})

	t.Run("register duplicate returns 409", func(t *testing.T) {
		api := newTestAPI(t)
		api.do(t, http.MethodPost, "/api/v1/auth/register", "", credentials)
		rec := api.do(t, http.MethodPost, "/api/v1/auth/register", "", credentials)
		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", rec.Code)
		}
	})

	t.Run("register malformed JSON returns 400", func(t *testing.T) {
		api := newTestAPI(t)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString("{not json"))
		rec := httptest.NewRecorder()
		api.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("login happy path", func(t *testing.T) {
		api := newTestAPI(t)
		api.do(t, http.MethodPost, "/api/v1/auth/register", "", credentials)
		rec := api.do(t, http.MethodPost, "/api/v1/auth/login", "", credentials)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("login wrong password returns 401", func(t *testing.T) {
		api := newTestAPI(t)
		api.do(t, http.MethodPost, "/api/v1/auth/register", "", credentials)
		rec := api.do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
			"email": "rider@example.com", "password": "wrong-password",
		})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
}

func TestProtectedRoutesRequireToken(t *testing.T) {
	api := newTestAPI(t)

	t.Run("missing token", func(t *testing.T) {
		if rec := api.do(t, http.MethodGet, "/api/v1/users/me/locations", "", nil); rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		if rec := api.do(t, http.MethodGet, "/api/v1/users/me/locations", "garbage", nil); rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("valid token", func(t *testing.T) {
		if rec := api.do(t, http.MethodGet, "/api/v1/users/me/locations", api.tokenFor(t), nil); rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})
}

func TestLocationHandlers(t *testing.T) {
	t.Run("search validation", func(t *testing.T) {
		api := newTestAPI(t)
		if rec := api.do(t, http.MethodGet, "/api/v1/locations/search?q=B", "", nil); rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("search success", func(t *testing.T) {
		api := newTestAPI(t)
		rec := api.do(t, http.MethodGet, "/api/v1/locations/search?q=Berlin", "", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		resp := decode[struct {
			Items []domain.Location `json:"items"`
		}](t, rec)
		if len(resp.Items) != 1 || resp.Items[0].ID == 0 {
			t.Fatalf("unexpected items: %+v", resp.Items)
		}
	})

	t.Run("add list and delete flow", func(t *testing.T) {
		api := newTestAPI(t)
		token := api.tokenFor(t)

		rec := api.do(t, http.MethodPost, "/api/v1/users/me/locations", token, map[string]any{
			"name": "Paris", "latitude": 48.85, "longitude": 2.35, "countryCode": "FR",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("add status = %d, want 201: %s", rec.Code, rec.Body.String())
		}
		saved := decode[domain.SavedLocation](t, rec)

		listRec := api.do(t, http.MethodGet, "/api/v1/users/me/locations", token, nil)
		list := decode[struct {
			Limit int                    `json:"limit"`
			Items []domain.SavedLocation `json:"items"`
		}](t, listRec)
		if list.Limit != service.MaxSavedLocations || len(list.Items) != 1 {
			t.Fatalf("unexpected list: %+v", list)
		}

		delRec := api.do(t, http.MethodDelete, "/api/v1/users/me/locations/"+itoa(saved.LocationID), token, nil)
		if delRec.Code != http.StatusNoContent {
			t.Fatalf("delete status = %d, want 204", delRec.Code)
		}
		missing := api.do(t, http.MethodDelete, "/api/v1/users/me/locations/"+itoa(saved.LocationID), token, nil)
		if missing.Code != http.StatusNotFound {
			t.Fatalf("delete missing status = %d, want 404", missing.Code)
		}
	})

	t.Run("sixth city returns 422", func(t *testing.T) {
		api := newTestAPI(t)
		token := api.tokenFor(t)
		for i := 0; i < service.MaxSavedLocations; i++ {
			rec := api.do(t, http.MethodPost, "/api/v1/users/me/locations", token, map[string]any{
				"name": "City", "latitude": float64(i), "longitude": float64(i),
			})
			if rec.Code != http.StatusCreated {
				t.Fatalf("seed %d status = %d, want 201", i, rec.Code)
			}
		}
		rec := api.do(t, http.MethodPost, "/api/v1/users/me/locations", token, map[string]any{
			"name": "Sixth", "latitude": 50.0, "longitude": 50.0,
		})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", rec.Code)
		}
	})
}

func TestRankHandler(t *testing.T) {
	api := newTestAPI(t)
	token := api.tokenFor(t)

	body := map[string]any{
		"cities": []map[string]any{
			{"name": "Berlin", "latitude": 52.0, "longitude": 13.0},
			{"name": "Madrid", "latitude": 40.0, "longitude": -3.0},
		},
		"criteria": map[string]any{
			"temperature":   map[string]any{"enabled": true, "weight": 1, "min": 18, "max": 26},
			"precipitation": map[string]any{"enabled": true, "weight": 1, "maxProbability": 20},
			"wind":          map[string]any{"enabled": true, "weight": 1, "min": 1, "max": 6},
		},
	}
	rec := api.do(t, http.MethodPost, "/api/v1/weather/rank", token, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	result := decode[domain.RankingResult](t, rec)
	if len(result.Cities) != 2 || !result.Cities[0].IsLeader || result.Cities[0].Name != "Madrid" {
		t.Fatalf("unexpected ranking: %+v", result.Cities)
	}

	t.Run("malformed body returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/weather/rank", bytes.NewBufferString("{"))
		req.Header.Set("Authorization", "Bearer "+token)
		r := httptest.NewRecorder()
		api.router.ServeHTTP(r, req)
		if r.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", r.Code)
		}
	})
}

func TestWindsurfHandlers(t *testing.T) {
	api := newTestAPI(t)
	token := api.tokenFor(t)

	putRec := api.do(t, http.MethodPut, "/api/v1/users/me/windsurf-profile", token, map[string]any{
		"level": "intermediate", "windSpeedMin": 6, "windSpeedMax": 14,
	})
	if putRec.Code != http.StatusOK {
		t.Fatalf("put status = %d, want 200: %s", putRec.Code, putRec.Body.String())
	}

	getRec := api.do(t, http.MethodGet, "/api/v1/users/me/windsurf-profile", token, nil)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", getRec.Code)
	}
	profile := decode[domain.WindsurfProfile](t, getRec)
	if profile.Level != domain.LevelIntermediate {
		t.Fatalf("unexpected profile: %+v", profile)
	}

	shore := 90.0
	recRec := api.do(t, http.MethodPost, "/api/v1/windsurf/recommendations", token, map[string]any{
		"days": 1, "slotHours": 3,
		"spots": []map[string]any{
			{"name": "Tarifa", "latitude": 36.0, "longitude": -5.6, "shoreNormalDeg": shore},
		},
	})
	if recRec.Code != http.StatusOK {
		t.Fatalf("recommend status = %d, want 200: %s", recRec.Code, recRec.Body.String())
	}
	result := decode[domain.RecommendationResult](t, recRec)
	if len(result.Recommendations) != 1 || !result.Recommendations[0].IsBest {
		t.Fatalf("unexpected recommendations: %+v", result.Recommendations)
	}
}

func TestCORSPreflight(t *testing.T) {
	api := newTestAPI(t)
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)

	rec := httptest.NewRecorder()
	api.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("Allow-Origin = %q", got)
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("expected Access-Control-Allow-Methods header")
	}
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}
