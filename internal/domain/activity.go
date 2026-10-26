package domain

import (
	"strings"
	"time"
)

type ActivityType string

const (
	ActivityRunning ActivityType = "running"
	ActivityCycling ActivityType = "cycling"
	ActivityWalking ActivityType = "walking"
	ActivityCustom  ActivityType = "custom"
)

type ActivityConditions struct {
	TempMin                     float64 `json:"tempMin"`
	TempMax                     float64 `json:"tempMax"`
	MaxPrecipitationProbability float64 `json:"maxPrecipitationProbability"`
	MaxWindSpeed                float64 `json:"maxWindSpeed"`
}

func DefaultConditions(activity string) ActivityConditions {
	switch strings.ToLower(strings.TrimSpace(activity)) {
	case "running", "бег":
		return ActivityConditions{
			TempMin:                     10.0,
			TempMax:                     20.0,
			MaxPrecipitationProbability: 20.0,
			MaxWindSpeed:                8.0,
		}
	case "cycling", "велосипед", "велоспорт":
		return ActivityConditions{
			TempMin:                     15.0,
			TempMax:                     25.0,
			MaxPrecipitationProbability: 10.0,
			MaxWindSpeed:                6.0,
		}
	case "walking", "прогулка", "ходьба":
		return ActivityConditions{
			TempMin:                     12.0,
			TempMax:                     22.0,
			MaxPrecipitationProbability: 30.0,
			MaxWindSpeed:                10.0,
		}
	default:
		return ActivityConditions{
			TempMin:                     15.0,
			TempMax:                     22.0,
			MaxPrecipitationProbability: 20.0,
			MaxWindSpeed:                8.0,
		}
	}
}

type ActivityTimeWindow struct {
	IsBest                      bool      `json:"isBest"`
	MatchesConditions           bool      `json:"matchesConditions"`
	Start                       time.Time `json:"start"`
	End                         time.Time `json:"end"`
	Score                       float64   `json:"score"`
	AverageTemperature          float64   `json:"averageTemperature"`
	AverageFeelsLike            float64   `json:"averageFeelsLike"`
	MaxPrecipitationProbability float64   `json:"maxPrecipitationProbability"`
	AverageWindSpeed            float64   `json:"averageWindSpeed"`
	WeatherCode                 int       `json:"weatherCode"`
	Condition                   string    `json:"condition"`
	Reasons                     []string  `json:"reasons"`
}

type ActivityRecommendationResult struct {
	GeneratedAt time.Time            `json:"generatedAt"`
	Location    Location             `json:"location"`
	Date        string               `json:"date"`
	Activity    string               `json:"activity"`
	Conditions  ActivityConditions   `json:"conditions"`
	Windows     []ActivityTimeWindow `json:"windows"`
	Message     string               `json:"message,omitempty"`
}
