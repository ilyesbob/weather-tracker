// Package weather contains the Open-Meteo adapters: a forecast client, a
// geocoding client and a caching decorator. All outbound calls are bounded by
// the configured timeout and never panic; errors are wrapped with context.
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"weathertracker/internal/domain"
)

// Provider request defaults and parsing constants.
const (
	minForecastDays       = 1
	defaultGeocodingLimit = 10
	errorBodyLimit        = 512 // bytes of the upstream error body kept in the message
	openMeteoTimeLayout   = "2006-01-02T15:04"
)

// Client implements both the WeatherProvider and Geocoder contracts using the
// public Open-Meteo API (https://open-meteo.com/).
type Client struct {
	forecastBase  string
	geocodingBase string
	http          *http.Client
}

// NewClient builds an Open-Meteo client. timeout bounds every request.
func NewClient(forecastBase, geocodingBase string, timeout time.Duration) *Client {
	return &Client{
		forecastBase:  forecastBase,
		geocodingBase: geocodingBase,
		http:          &http.Client{Timeout: timeout},
	}
}

type forecastResponse struct {
	Current struct {
		Time          string  `json:"time"`
		Temperature   float64 `json:"temperature_2m"`
		FeelsLike     float64 `json:"apparent_temperature"`
		Humidity      float64 `json:"relative_humidity_2m"`
		WindSpeed     float64 `json:"wind_speed_10m"`
		WindDirection float64 `json:"wind_direction_10m"`
		WindGusts     float64 `json:"wind_gusts_10m"`
		WeatherCode   int     `json:"weather_code"`
	} `json:"current"`
	Hourly struct {
		Time                     []string  `json:"time"`
		Temperature              []float64 `json:"temperature_2m"`
		FeelsLike                []float64 `json:"apparent_temperature"`
		Humidity                 []float64 `json:"relative_humidity_2m"`
		PrecipitationProbability []float64 `json:"precipitation_probability"`
		WindSpeed                []float64 `json:"wind_speed_10m"`
		WindDirection            []float64 `json:"wind_direction_10m"`
		WindGusts                []float64 `json:"wind_gusts_10m"`
		WeatherCode              []int     `json:"weather_code"`
	} `json:"hourly"`
}

// Fetch requests the current weather and hourly forecast for a coordinate.
func (c *Client) Fetch(ctx context.Context, latitude, longitude float64, forecastDays int) (domain.WeatherSnapshot, error) {
	if forecastDays < minForecastDays {
		forecastDays = minForecastDays
	}
	const hourlyVars = "temperature_2m,apparent_temperature,relative_humidity_2m," +
		"precipitation_probability,wind_speed_10m,wind_direction_10m,wind_gusts_10m,weather_code"

	q := url.Values{}
	q.Set("latitude", strconv.FormatFloat(latitude, 'f', 4, 64))
	q.Set("longitude", strconv.FormatFloat(longitude, 'f', 4, 64))
	q.Set("current", "temperature_2m,apparent_temperature,relative_humidity_2m,"+
		"wind_speed_10m,wind_direction_10m,wind_gusts_10m,weather_code")
	q.Set("hourly", hourlyVars)
	q.Set("forecast_days", strconv.Itoa(forecastDays))
	q.Set("timezone", "UTC")

	endpoint := c.forecastBase + "/v1/forecast?" + q.Encode()
	var raw forecastResponse
	if err := c.getJSON(ctx, endpoint, &raw); err != nil {
		return domain.WeatherSnapshot{}, fmt.Errorf("fetch forecast (%.3f,%.3f): %w", latitude, longitude, err)
	}

	snap := domain.WeatherSnapshot{
		Latitude:  latitude,
		Longitude: longitude,
		FetchedAt: time.Now().UTC(),
		Current: domain.CurrentWeather{
			Temperature:   raw.Current.Temperature,
			FeelsLike:     raw.Current.FeelsLike,
			Humidity:      int(raw.Current.Humidity),
			WindSpeed:     raw.Current.WindSpeed,
			WindDirection: raw.Current.WindDirection,
			WindGusts:     raw.Current.WindGusts,
			WeatherCode:   raw.Current.WeatherCode,
			Condition:     conditionFromCode(raw.Current.WeatherCode),
			Icon:          iconFromCode(raw.Current.WeatherCode),
			ObservedAt:    parseTime(raw.Current.Time),
		},
	}
	snap.Hourly = buildHourly(raw)
	return snap, nil
}

func buildHourly(raw forecastResponse) []domain.HourlyForecast {
	n := len(raw.Hourly.Time)
	out := make([]domain.HourlyForecast, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, domain.HourlyForecast{
			Time:                     parseTime(raw.Hourly.Time[i]),
			Temperature:              at(raw.Hourly.Temperature, i),
			FeelsLike:                at(raw.Hourly.FeelsLike, i),
			Humidity:                 int(at(raw.Hourly.Humidity, i)),
			PrecipitationProbability: at(raw.Hourly.PrecipitationProbability, i),
			WindSpeed:                at(raw.Hourly.WindSpeed, i),
			WindDirection:            at(raw.Hourly.WindDirection, i),
			WindGusts:                at(raw.Hourly.WindGusts, i),
			WeatherCode:              atInt(raw.Hourly.WeatherCode, i),
		})
	}
	return out
}

type geocodingResponse struct {
	Results []struct {
		ID          int64   `json:"id"`
		Name        string  `json:"name"`
		Latitude    float64 `json:"latitude"`
		Longitude   float64 `json:"longitude"`
		CountryCode string  `json:"country_code"`
		Timezone    string  `json:"timezone"`
	} `json:"results"`
}

// Search resolves a free-text query to candidate locations via Open-Meteo
// geocoding. It is used by the location search endpoint.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]domain.Location, error) {
	if limit <= 0 {
		limit = defaultGeocodingLimit
	}
	q := url.Values{}
	q.Set("name", query)
	q.Set("count", strconv.Itoa(limit))
	q.Set("language", "en")
	q.Set("format", "json")

	endpoint := c.geocodingBase + "/v1/search?" + q.Encode()
	var raw geocodingResponse
	if err := c.getJSON(ctx, endpoint, &raw); err != nil {
		return nil, fmt.Errorf("geocode %q: %w", query, err)
	}

	out := make([]domain.Location, 0, len(raw.Results))
	for _, r := range raw.Results {
		out = append(out, domain.Location{
			Name:        r.Name,
			Latitude:    r.Latitude,
			Longitude:   r.Longitude,
			CountryCode: r.CountryCode,
			Timezone:    r.Timezone,
		})
	}
	return out, nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, errorBodyLimit))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func at(s []float64, i int) float64 {
	if i >= 0 && i < len(s) {
		return s[i]
	}
	return 0
}

func atInt(s []int, i int) int {
	if i >= 0 && i < len(s) {
		return s[i]
	}
	return 0
}

func parseTime(v string) time.Time {
	if v == "" {
		return time.Time{}
	}
	t, err := time.Parse(openMeteoTimeLayout, v)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}
