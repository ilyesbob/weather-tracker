package service

import (
	"fmt"
	"math"
	"time"

	"weathertracker/internal/domain"
)

// Metric names are part of the public API contract: the React client switches
// on them to render the per-criterion breakdown, so they are declared once and
// reused between evaluation and weighting instead of being duplicated strings.
const (
	metricTemperature   = "temperature"
	metricPrecipitation = "precipitation"
	metricWind          = "wind"
)

// Score bounds and shaping factors for rangeScore. They are named so that
// tuning the ranking behaviour does not require reverse-engineering literals.
const (
	scoreCeiling          = 100.0 // ideal point or a value inside a band without an ideal
	scoreFloorWithinBand  = 70.0  // lowest score still treated as "inside the comfort band"
	idealDeviationPenalty = 30.0  // points removed at the far edge of the band when an ideal is set
)

// Approximate precipitation severity in percent, derived from WMO weather codes
// (https://open-meteo.com/en/docs). Severity is a provider-independent proxy so
// the client always receives a comparable numeric metric.
const (
	precipNone    = 0.0
	precipLight   = 10.0
	precipFog     = 20.0
	precipDrizzle = 40.0
	precipSnow    = 70.0
	precipRain    = 80.0
	precipShowers = 85.0
	precipThunder = 95.0
	precipUnknown = 50.0
)

// evaluateCriteria turns the user's priorities into a per-metric score
// breakdown. Keeping the breakdown explicit makes the ranking explainable
// rather than a black box.
func evaluateCriteria(w domain.CurrentWeather, c domain.RankingCriteria) []domain.MetricScore {
	metrics := make([]domain.MetricScore, 0, 3)

	if c.Temperature.Enabled {
		metrics = append(metrics, domain.MetricScore{
			Metric: metricTemperature,
			Value:  round1(w.Temperature),
			Score:  round1(rangeScore(w.Temperature, c.Temperature.Min, c.Temperature.Max, c.Temperature.Ideal)),
			Reason: fmt.Sprintf("Temperature is %.1f°C (comfort band %.0f–%.0f°C).",
				w.Temperature, c.Temperature.Min, c.Temperature.Max),
		})
	}

	if c.Precipitation.Enabled {
		metrics = append(metrics, domain.MetricScore{
			Metric: metricPrecipitation,
			Value:  precipitationValue(w),
			Score:  round1(precipitationScore(w)),
			Reason: fmt.Sprintf("%s (weather code %d).", w.Condition, w.WeatherCode),
		})
	}

	if c.Wind.Enabled {
		metrics = append(metrics, domain.MetricScore{
			Metric: metricWind,
			Value:  round1(w.WindSpeed),
			Score:  round1(rangeScore(w.WindSpeed, c.Wind.Min, c.Wind.Max, c.Wind.Ideal)),
			Reason: fmt.Sprintf("Wind speed is %.1f m/s (comfort band %.0f–%.0f m/s).",
				w.WindSpeed, c.Wind.Min, c.Wind.Max),
		})
	}

	return metrics
}

// weightedScore returns the weight-aware average of the metric scores scaled to
// 0–100. The per-metric weights come straight from the user's criteria.
func weightedScore(metrics []domain.MetricScore, c domain.RankingCriteria) float64 {
	if len(metrics) == 0 {
		return 0
	}
	weights := map[string]float64{
		metricTemperature:   c.Temperature.Weight,
		metricPrecipitation: c.Precipitation.Weight,
		metricWind:          c.Wind.Weight,
	}

	var weighted, totalWeight float64
	for _, m := range metrics {
		w := weights[m.Metric]
		if w <= 0 {
			w = 1 // an enabled metric without an explicit weight counts once
		}
		weighted += m.Score * w
		totalWeight += w
	}
	if totalWeight == 0 {
		return 0
	}
	return round1(weighted / totalWeight)
}

// rangeScore scores a numeric value inside [min, max]. Values inside the band
// score between scoreFloorWithinBand and scoreCeiling (scoreCeiling at the
// ideal value); values outside degrade linearly with the distance from the
// nearest bound, so the score stays monotonic and predictable for the client.
func rangeScore(v, min, max float64, ideal *float64) float64 {
	span := max - min
	if span <= 0 {
		span = 1
	}
	switch {
	case v < min:
		return clamp(scoreCeiling-(min-v)/span*scoreCeiling, 0, scoreCeiling)
	case v > max:
		return clamp(scoreCeiling-(v-max)/span*scoreCeiling, 0, scoreCeiling)
	}
	if ideal == nil {
		return scoreCeiling
	}
	half := span / 2
	if half <= 0 {
		return scoreCeiling
	}
	return clamp(scoreCeiling-math.Abs(v-*ideal)/half*idealDeviationPenalty, scoreFloorWithinBand, scoreCeiling)
}

// precipitationValue maps the WMO code to an approximate severity in percent.
func precipitationValue(w domain.CurrentWeather) float64 {
	switch {
	case w.WeatherCode == 0:
		return precipNone
	case w.WeatherCode <= 3:
		return precipLight
	case w.WeatherCode == 45 || w.WeatherCode == 48:
		return precipFog
	case w.WeatherCode >= 51 && w.WeatherCode <= 57:
		return precipDrizzle
	case w.WeatherCode >= 61 && w.WeatherCode <= 67:
		return precipRain
	case w.WeatherCode >= 71 && w.WeatherCode <= 77:
		return precipSnow
	case w.WeatherCode >= 80 && w.WeatherCode <= 82:
		return precipShowers
	case w.WeatherCode >= 95:
		return precipThunder
	default:
		return precipUnknown
	}
}

func precipitationScore(w domain.CurrentWeather) float64 {
	return clamp(scoreCeiling-precipitationValue(w), 0, scoreCeiling)
}

// summarize builds the human-readable justification list. The staleness note
// embeds the configured threshold instead of a hard-coded "5 minutes" so the
// message stays truthful when STALE_THRESHOLD is overridden.
func summarize(metrics []domain.MetricScore, stale bool, staleThreshold time.Duration) []string {
	reasons := make([]string, 0, len(metrics)+1)
	for _, m := range metrics {
		reasons = append(reasons, m.Reason)
	}
	if stale {
		reasons = append(reasons, fmt.Sprintf("Last known data is older than %s (stale).", staleThreshold))
	}
	return reasons
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

// forecastWeightedScore evaluates the criteria hour by hour over the upcoming
// horizon and returns the average weighted score. It lets the ranking reflect
// the near-term forecast (User Story 1: "анализирует текущую погоду и прогноз")
// instead of a single instant. The second return value is false when no future
// hours are available, so callers can fall back to the current-conditions score.
func forecastWeightedScore(hourly []domain.HourlyForecast, c domain.RankingCriteria, now time.Time, horizonHours int) (float64, bool) {
	window := upcomingHours(hourly, now, horizonHours)
	if len(window) == 0 {
		return 0, false
	}
	var sum float64
	for _, h := range window {
		// Reuse the same evaluation path as the current observation so the
		// forecast and current scores are directly comparable.
		forecast := domain.CurrentWeather{
			Temperature:   h.Temperature,
			FeelsLike:     h.FeelsLike,
			Humidity:      h.Humidity,
			WindSpeed:     h.WindSpeed,
			WindDirection: h.WindDirection,
			WindGusts:     h.WindGusts,
			WeatherCode:   h.WeatherCode,
		}
		sum += weightedScore(evaluateCriteria(forecast, c), c)
	}
	return round1(sum / float64(len(window))), true
}

// upcomingHours returns up to horizonHours future forecast entries, skipping
// hours that already started.
func upcomingHours(hourly []domain.HourlyForecast, now time.Time, horizonHours int) []domain.HourlyForecast {
	out := make([]domain.HourlyForecast, 0, horizonHours)
	for _, h := range hourly {
		if h.Time.Before(now) {
			continue
		}
		out = append(out, h)
		if len(out) >= horizonHours {
			break
		}
	}
	return out
}
