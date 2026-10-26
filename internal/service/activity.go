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

const (
	defaultActivityDurationHrs = 1
	maxActivityDurationHrs     = 4
	maxActivityForecastDays    = 7
	maxActivityResults         = 5

	msgNoUpcomingHours   = "No upcoming forecast hours available for the selected date."
	msgNoActivityMatch   = "No slot fully matches the requested conditions; showing the closest available options."
	msgActivityStaleData = "Forecast data is stale; recommendations require up-to-date forecast."
)

type ActivityService struct {
	catalog        repository.LocationRepository
	saved          repository.SavedLocationRepository
	weather        repository.WeatherProvider
	staleThreshold time.Duration
	now            func() time.Time
}

func NewActivityService(
	catalog repository.LocationRepository,
	saved repository.SavedLocationRepository,
	weather repository.WeatherProvider,
	staleThreshold time.Duration,
) *ActivityService {
	return &ActivityService{
		catalog:        catalog,
		saved:          saved,
		weather:        weather,
		staleThreshold: staleThreshold,
		now:            time.Now,
	}
}

type ActivityInput struct {
	LocationID    *int64                     `json:"locationId,omitempty"`
	City          *CityInput                 `json:"city,omitempty"`
	Date          string                     `json:"date"`
	Activity      string                     `json:"activity"`
	Conditions    *domain.ActivityConditions `json:"conditions,omitempty"`
	DurationHours int                        `json:"durationHours,omitempty"`
}

func (s *ActivityService) Recommend(ctx context.Context, userID string, in ActivityInput) (domain.ActivityRecommendationResult, error) {
	now := s.now().UTC()

	loc, err := s.resolveLocation(ctx, userID, in)
	if err != nil {
		return domain.ActivityRecommendationResult{}, err
	}

	targetDate, err := time.Parse("2006-01-02", strings.TrimSpace(in.Date))
	if err != nil {
		return domain.ActivityRecommendationResult{}, fmt.Errorf("%w: invalid date format %q, expected YYYY-MM-DD", domain.ErrValidation, in.Date)
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if targetDate.Before(today) {
		return domain.ActivityRecommendationResult{}, fmt.Errorf("%w: date cannot be in the past", domain.ErrValidation)
	}

	daysAhead := int(targetDate.Sub(today).Hours()/24) + 1
	if daysAhead > maxActivityForecastDays {
		return domain.ActivityRecommendationResult{}, fmt.Errorf("%w: date is beyond maximum forecast horizon of %d days", domain.ErrValidation, maxActivityForecastDays)
	}

	cond := s.resolveConditions(in)
	if err := validateConditions(cond); err != nil {
		return domain.ActivityRecommendationResult{}, err
	}

	duration := clampInt(in.DurationHours, defaultActivityDurationHrs, 1, maxActivityDurationHrs)

	snapshot, err := s.weather.Fetch(ctx, loc.Latitude, loc.Longitude, daysAhead)
	if err != nil {
		return domain.ActivityRecommendationResult{}, fmt.Errorf("fetch weather: %w", err)
	}

	if snapshot.Stale(now, s.staleThreshold) {
		return domain.ActivityRecommendationResult{}, fmt.Errorf("%w: %s", domain.ErrValidation, msgActivityStaleData)
	}

	dayHours := filterHoursForDate(snapshot.Hourly, targetDate, now)
	if len(dayHours) < duration {
		return domain.ActivityRecommendationResult{
			GeneratedAt: now,
			Location:    loc,
			Date:        in.Date,
			Activity:    in.Activity,
			Conditions:  cond,
			Message:     msgNoUpcomingHours,
		}, nil
	}

	windows, anyAcceptable := scoreWindows(dayHours, duration, cond)
	if len(windows) == 0 {
		return domain.ActivityRecommendationResult{
			GeneratedAt: now,
			Location:    loc,
			Date:        in.Date,
			Activity:    in.Activity,
			Conditions:  cond,
			Message:     msgNoUpcomingHours,
		}, nil
	}

	resultWindows := selectWindows(windows, anyAcceptable)

	result := domain.ActivityRecommendationResult{
		GeneratedAt: now,
		Location:    loc,
		Date:        in.Date,
		Activity:    in.Activity,
		Conditions:  cond,
		Windows:     resultWindows,
	}

	if !anyAcceptable {
		result.Message = msgNoActivityMatch
	}

	return result, nil
}

func (s *ActivityService) resolveLocation(ctx context.Context, userID string, in ActivityInput) (domain.Location, error) {
	if in.LocationID != nil && *in.LocationID > 0 {
		loc, err := s.catalog.ByID(ctx, *in.LocationID)
		if err == nil {
			return loc, nil
		}
	}

	if in.City != nil && strings.TrimSpace(in.City.Name) != "" {
		return domain.Location{
			Name:        strings.TrimSpace(in.City.Name),
			Latitude:    in.City.Latitude,
			Longitude:   in.City.Longitude,
			CountryCode: in.City.CountryCode,
		}, nil
	}

	if userID != "" {
		saved, err := s.saved.List(ctx, userID)
		if err == nil && len(saved) > 0 {
			first := saved[0]
			return domain.Location{
				ID:          first.LocationID,
				Name:        first.Name,
				Latitude:    first.Latitude,
				Longitude:   first.Longitude,
				CountryCode: first.CountryCode,
				Timezone:    first.Timezone,
			}, nil
		}
	}

	return domain.Location{}, fmt.Errorf("%w: locationId or city coordinates are required", domain.ErrValidation)
}

func (s *ActivityService) resolveConditions(in ActivityInput) domain.ActivityConditions {
	if in.Conditions != nil && (in.Conditions.TempMin != 0 || in.Conditions.TempMax != 0 ||
		in.Conditions.MaxPrecipitationProbability != 0 || in.Conditions.MaxWindSpeed != 0) {
		return *in.Conditions
	}
	return domain.DefaultConditions(in.Activity)
}

func validateConditions(c domain.ActivityConditions) error {
	if c.TempMin > c.TempMax {
		return fmt.Errorf("%w: tempMin (%.1f) cannot be greater than tempMax (%.1f)", domain.ErrValidation, c.TempMin, c.TempMax)
	}
	if c.MaxPrecipitationProbability < 0 || c.MaxPrecipitationProbability > 100 {
		return fmt.Errorf("%w: maxPrecipitationProbability must be between 0 and 100", domain.ErrValidation)
	}
	if c.MaxWindSpeed < 0 {
		return fmt.Errorf("%w: maxWindSpeed cannot be negative", domain.ErrValidation)
	}
	return nil
}

func filterHoursForDate(hourly []domain.HourlyForecast, targetDate, now time.Time) []domain.HourlyForecast {
	targetY, targetM, targetD := targetDate.Date()
	nowY, nowM, nowD := now.Date()
	isToday := targetY == nowY && targetM == nowM && targetD == nowD

	out := make([]domain.HourlyForecast, 0, 24)
	for _, h := range hourly {
		hTime := h.Time.UTC()
		y, m, d := hTime.Date()
		if y == targetY && m == targetM && d == targetD {
			if isToday && hTime.Before(now) {
				continue
			}
			out = append(out, h)
		}
	}
	return out
}

type scoredActivityWindow struct {
	window     domain.ActivityTimeWindow
	acceptable bool
}

func scoreWindows(hours []domain.HourlyForecast, duration int, cond domain.ActivityConditions) ([]scoredActivityWindow, bool) {
	windows := make([]scoredActivityWindow, 0)
	var anyAcceptable bool

	for i := 0; i+duration <= len(hours); i++ {
		slot := hours[i : i+duration]

		temps := make([]float64, 0, duration)
		feels := make([]float64, 0, duration)
		precips := make([]float64, 0, duration)
		winds := make([]float64, 0, duration)

		for _, h := range slot {
			temps = append(temps, h.Temperature)
			feels = append(feels, h.FeelsLike)
			precips = append(precips, h.PrecipitationProbability)
			winds = append(winds, h.WindSpeed)
		}

		avgTemp := round1(mean(temps))
		avgFeels := round1(mean(feels))
		maxPrecip := round1(maxFloatSlice(precips))
		avgWind := round1(mean(winds))
		midCode := slot[len(slot)/2].WeatherCode

		tempOK := avgTemp >= cond.TempMin && avgTemp <= cond.TempMax
		precipOK := maxPrecip <= cond.MaxPrecipitationProbability
		windOK := avgWind <= cond.MaxWindSpeed
		acceptable := tempOK && precipOK && windOK
		if acceptable {
			anyAcceptable = true
		}

		idealTemp := (cond.TempMin + cond.TempMax) / 2
		tempScore := rangeScore(avgTemp, cond.TempMin, cond.TempMax, &idealTemp)

		var precipScore float64
		if maxPrecip <= cond.MaxPrecipitationProbability {
			span := cond.MaxPrecipitationProbability
			if span <= 0 {
				precipScore = scoreCeiling
			} else {
				precipScore = scoreCeiling - (maxPrecip / span * 30.0)
			}
		} else {
			span := 100.0 - cond.MaxPrecipitationProbability
			if span <= 0 {
				span = 1.0
			}
			precipScore = clamp(scoreFloorWithinBand-((maxPrecip-cond.MaxPrecipitationProbability)/span*scoreFloorWithinBand), 0, scoreFloorWithinBand)
		}

		var windScore float64
		if avgWind <= cond.MaxWindSpeed {
			span := cond.MaxWindSpeed
			if span <= 0 {
				windScore = scoreCeiling
			} else {
				windScore = scoreCeiling - (avgWind / span * 30.0)
			}
		} else {
			span := cond.MaxWindSpeed
			if span <= 0 {
				span = 1.0
			}
			windScore = clamp(scoreFloorWithinBand-((avgWind-cond.MaxWindSpeed)/span*scoreFloorWithinBand), 0, scoreFloorWithinBand)
		}

		totalScore := round1(clamp(0.40*tempScore+0.35*precipScore+0.25*windScore, 0, scoreCeiling))

		reasons := []string{
			fmt.Sprintf("Temperature is %.1f°C (comfort band %.0f–%.0f°C).", avgTemp, cond.TempMin, cond.TempMax),
			fmt.Sprintf("Precipitation probability is %.0f%% (max allowed %.0f%%).", maxPrecip, cond.MaxPrecipitationProbability),
			fmt.Sprintf("Wind speed is %.1f m/s (max allowed %.1f m/s).", avgWind, cond.MaxWindSpeed),
		}
		if acceptable {
			reasons = append(reasons, "All conditions match the requested criteria.")
		} else {
			reasons = append(reasons, "Conditions deviate from requested criteria, but this is the closest available time slot.")
		}

		win := domain.ActivityTimeWindow{
			MatchesConditions:           acceptable,
			Start:                       slot[0].Time,
			End:                         slot[len(slot)-1].Time.Add(time.Hour),
			Score:                       totalScore,
			AverageTemperature:          avgTemp,
			AverageFeelsLike:            avgFeels,
			MaxPrecipitationProbability: maxPrecip,
			AverageWindSpeed:            avgWind,
			WeatherCode:                 midCode,
			Condition:                   wmoCondition(midCode),
			Reasons:                     reasons,
		}

		windows = append(windows, scoredActivityWindow{window: win, acceptable: acceptable})
	}

	return windows, anyAcceptable
}

func selectWindows(windows []scoredActivityWindow, anyAcceptable bool) []domain.ActivityTimeWindow {
	slices.SortFunc(windows, func(a, b scoredActivityWindow) int {
		if a.window.Score != b.window.Score {
			if a.window.Score > b.window.Score {
				return -1
			}
			return 1
		}
		return a.window.Start.Compare(b.window.Start)
	})

	out := make([]domain.ActivityTimeWindow, 0, maxActivityResults)
	if anyAcceptable {
		for _, w := range windows {
			if w.acceptable {
				out = append(out, w.window)
				if len(out) >= maxActivityResults {
					break
				}
			}
		}
	} else {
		for _, w := range windows {
			out = append(out, w.window)
			if len(out) >= 3 {
				break
			}
		}
	}

	if len(out) > 0 {
		out[0].IsBest = true
	}
	return out
}

func maxFloatSlice(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func wmoCondition(code int) string {
	switch code {
	case 0:
		return "Clear sky"
	case 1:
		return "Mainly clear"
	case 2:
		return "Partly cloudy"
	case 3:
		return "Overcast"
	case 45, 48:
		return "Fog"
	case 51, 53, 55:
		return "Drizzle"
	case 56, 57:
		return "Freezing drizzle"
	case 61, 63, 65:
		return "Rain"
	case 66, 67:
		return "Freezing rain"
	case 71, 73, 75, 77:
		return "Snow"
	case 80, 81, 82:
		return "Rain showers"
	case 85, 86:
		return "Snow showers"
	case 95:
		return "Thunderstorm"
	case 96, 99:
		return "Thunderstorm with hail"
	default:
		return "Unknown"
	}
}
