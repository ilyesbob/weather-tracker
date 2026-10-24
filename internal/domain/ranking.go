package domain

import "time"

// MetricPreference describes a user's weighting for a single numeric weather
// metric (temperature or wind). Weight scales the metric's contribution to the
// final match score, while [Min, Max] defines the acceptable comfort band and
// Ideal (optional) the most desirable value.
type MetricPreference struct {
	Enabled bool     `json:"enabled"`
	Weight  float64  `json:"weight"`
	Min     float64  `json:"min"`
	Max     float64  `json:"max"`
	Ideal   *float64 `json:"ideal,omitempty"`
}

// PrecipitationCriterion captures the tolerance to rain: any probability above
// MaxProbability is penalised.
type PrecipitationCriterion struct {
	Enabled        bool    `json:"enabled"`
	Weight         float64 `json:"weight"`
	MaxProbability float64 `json:"maxProbability"`
}

// RankingCriteria is the complete set of priorities a traveller can express for
// User Story 1.
type RankingCriteria struct {
	Temperature   MetricPreference       `json:"temperature"`
	Wind          MetricPreference       `json:"wind"`
	Precipitation PrecipitationCriterion `json:"precipitation"`
}

// MetricScore is the per-metric breakdown shown to the user so the ranking is
// explainable rather than a black box.
type MetricScore struct {
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

// RankedCity is a single entry of the comparison result.
type RankedCity struct {
	Rank        int            `json:"rank"`
	IsLeader    bool           `json:"isLeader"`
	LocationID  int64          `json:"locationId"`
	Name        string         `json:"name"`
	CountryCode string         `json:"countryCode"`
	Latitude    float64        `json:"latitude"`
	Longitude   float64        `json:"longitude"`
	Score       float64        `json:"score"`
	IsStale     bool           `json:"isStale"`
	LastUpdated time.Time      `json:"lastUpdated"`
	Weather     CurrentWeather `json:"weather"`
	Metrics     []MetricScore  `json:"metrics"`
	Reasons     []string       `json:"reasons"`
}

// RankingResult is the payload returned by the ranking endpoint.
type RankingResult struct {
	GeneratedAt time.Time       `json:"generatedAt"`
	Criteria    RankingCriteria `json:"criteria"`
	Cities      []RankedCity    `json:"cities"`
}
