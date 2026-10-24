package service

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"weathertracker/internal/domain"
	"weathertracker/internal/repository"
)

// rankingForecastDays limits the Open-Meteo call to the current day: ranking
// only needs present conditions plus the near-term forecast, which keeps
// responses fast and cache-friendly.
const rankingForecastDays = 1

// rankingForecastHorizon is how many upcoming hours are folded into the score
// together with the current observation, so a trip ranking reflects the near
// future rather than a single moment (User Story 1).
const rankingForecastHorizon = 12

// RankingService implements User Story 1: it scores each candidate city
// against the user's weather priorities and returns them in descending order.
type RankingService struct {
	catalog        repository.LocationRepository
	weather        repository.WeatherProvider
	staleThreshold time.Duration
	now            func() time.Time
}

// NewRankingService wires the ranking service dependencies.
func NewRankingService(catalog repository.LocationRepository, weather repository.WeatherProvider, staleThreshold time.Duration) *RankingService {
	return &RankingService{
		catalog:        catalog,
		weather:        weather,
		staleThreshold: staleThreshold,
		now:            time.Now,
	}
}

// CityInput describes an ad-hoc city supplied inline (not necessarily saved).
type CityInput struct {
	Name        string  `json:"name"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	CountryCode string  `json:"countryCode"`
}

// RankingInput is the ranking request body.
type RankingInput struct {
	LocationIDs []int64                `json:"locationIds"`
	Cities      []CityInput            `json:"cities"`
	Criteria    domain.RankingCriteria `json:"criteria"`
}

// Rank fetches the current weather and near-term forecast for every candidate,
// computes a weighted match score (current conditions blended with the next
// hours) and returns the sorted ranking. Cities whose data is old are still
// returned but flagged with IsStale = true.
func (s *RankingService) Rank(ctx context.Context, in RankingInput) (domain.RankingResult, error) {
	locations, err := s.resolve(ctx, in)
	if err != nil {
		return domain.RankingResult{}, err
	}
	if len(locations) == 0 {
		return domain.RankingResult{}, fmt.Errorf("%w: at least one city is required", domain.ErrValidation)
	}
	if len(locations) > MaxSavedLocations {
		return domain.RankingResult{}, fmt.Errorf("%w: a maximum of %d cities can be ranked", domain.ErrLimitExceeded, MaxSavedLocations)
	}

	now := s.now().UTC()
	cities := make([]domain.RankedCity, 0, len(locations))
	for _, loc := range locations {
		cities = append(cities, s.scoreCity(ctx, loc, in.Criteria, now))
	}
	sortCitiesByScore(cities)

	return domain.RankingResult{
		GeneratedAt: now,
		Criteria:    in.Criteria,
		Cities:      cities,
	}, nil
}

// sortCitiesByScore orders cities by descending score, breaking ties
// alphabetically, and stamps the rank and leader flags. The deterministic order
// matters so the UI does not flicker between equivalent results.
func sortCitiesByScore(cities []domain.RankedCity) {
	slices.SortFunc(cities, func(a, b domain.RankedCity) int {
		if a.Score != b.Score {
			if a.Score > b.Score {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Name, b.Name)
	})
	for i := range cities {
		cities[i].Rank = i + 1
		cities[i].IsLeader = i == 0
	}
}

func (s *RankingService) resolve(ctx context.Context, in RankingInput) ([]domain.Location, error) {
	out := make([]domain.Location, 0, len(in.LocationIDs)+len(in.Cities))
	for _, id := range in.LocationIDs {
		loc, err := s.catalog.ByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("resolve location %d: %w", id, err)
		}
		out = append(out, loc)
	}
	for _, c := range in.Cities {
		if strings.TrimSpace(c.Name) == "" {
			return nil, fmt.Errorf("%w: city name is required", domain.ErrValidation)
		}
		out = append(out, domain.Location{
			Name:        strings.TrimSpace(c.Name),
			Latitude:    c.Latitude,
			Longitude:   c.Longitude,
			CountryCode: c.CountryCode,
		})
	}
	return out, nil
}

func (s *RankingService) scoreCity(ctx context.Context, loc domain.Location, criteria domain.RankingCriteria, now time.Time) domain.RankedCity {
	ranked := domain.RankedCity{
		LocationID:  loc.ID,
		Name:        loc.Name,
		CountryCode: loc.CountryCode,
		Latitude:    loc.Latitude,
		Longitude:   loc.Longitude,
	}

	snapshot, err := s.weather.Fetch(ctx, loc.Latitude, loc.Longitude, rankingForecastDays)
	if err != nil {
		ranked.Reasons = []string{"Weather data is currently unavailable for this city."}
		return ranked
	}

	ranked.IsStale = snapshot.Stale(now, s.staleThreshold)
	ranked.LastUpdated = snapshot.FetchedAt
	ranked.Weather = snapshot.Current
	ranked.Metrics = evaluateCriteria(snapshot.Current, criteria)
	score, usedForecast := blendedScore(ranked.Metrics, snapshot, criteria, now)
	ranked.Score = score
	ranked.Reasons = summarize(ranked.Metrics, ranked.IsStale, s.staleThreshold)
	if usedForecast {
		ranked.Reasons = append(ranked.Reasons,
			fmt.Sprintf("Score also blends the next %dh forecast.", rankingForecastHorizon))
	}
	return ranked
}

// blendedScore averages the current-conditions score with the forecast score.
// When no future hours are available (or the forecast is unavailable), it falls
// back to the current-conditions score so the ranking never becomes empty.
func blendedScore(currentMetrics []domain.MetricScore, snapshot domain.WeatherSnapshot, criteria domain.RankingCriteria, now time.Time) (float64, bool) {
	currentScore := weightedScore(currentMetrics, criteria)
	forecastScore, ok := forecastWeightedScore(snapshot.Hourly, criteria, now, rankingForecastHorizon)
	if !ok {
		return currentScore, false
	}
	return round1((currentScore + forecastScore) / 2), true
}
