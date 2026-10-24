package domain

import "time"

// CurrentWeather is a point-in-time observation for a location.
type CurrentWeather struct {
	Temperature   float64   `json:"temperature"`
	FeelsLike     float64   `json:"feelsLike"`
	Humidity      int       `json:"humidity"`
	WindSpeed     float64   `json:"windSpeed"`
	WindDirection float64   `json:"windDirection"`
	WindGusts     float64   `json:"windGusts"`
	WeatherCode   int       `json:"weatherCode"`
	Condition     string    `json:"condition"`
	Icon          string    `json:"icon"`
	ObservedAt    time.Time `json:"observedAt"`
}

// HourlyForecast is a single forecast hour for a location.
type HourlyForecast struct {
	Time                     time.Time `json:"time"`
	Temperature              float64   `json:"temperature"`
	FeelsLike                float64   `json:"feelsLike"`
	Humidity                 int       `json:"humidity"`
	PrecipitationProbability float64   `json:"precipitationProbability"`
	WindSpeed                float64   `json:"windSpeed"`
	WindDirection            float64   `json:"windDirection"`
	WindGusts                float64   `json:"windGusts"`
	WeatherCode              int       `json:"weatherCode"`
}

// WeatherSnapshot bundles the current observation with the hourly forecast for
// a coordinate. FetchedAt drives the staleness decision in the services: any
// snapshot older than the configured threshold is surfaced to the client with
// an explicit "stale data" marker instead of being silently dropped.
type WeatherSnapshot struct {
	Latitude  float64          `json:"latitude"`
	Longitude float64          `json:"longitude"`
	Current   CurrentWeather   `json:"current"`
	Hourly    []HourlyForecast `json:"hourly"`
	FetchedAt time.Time        `json:"fetchedAt"`
}

// Stale reports whether the snapshot was fetched more than threshold ago.
func (s WeatherSnapshot) Stale(at time.Time, threshold time.Duration) bool {
	return at.Sub(s.FetchedAt) > threshold
}
