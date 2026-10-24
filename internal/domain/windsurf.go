package domain

import "time"

// SkillLevel models the rider's experience for User Story 2. It influences how
// tolerant the recommendation is to offshore wind and strong gusts.
type SkillLevel string

// Supported skill levels, ordered from least to most experienced.
const (
	LevelBeginner     SkillLevel = "beginner"
	LevelIntermediate SkillLevel = "intermediate"
	LevelAdvanced     SkillLevel = "advanced"
	LevelPro          SkillLevel = "pro"
)

// Valid reports whether the level is one of the supported values.
func (l SkillLevel) Valid() bool {
	switch l {
	case LevelBeginner, LevelIntermediate, LevelAdvanced, LevelPro:
		return true
	default:
		return false
	}
}

// WindsurfProfile is the persisted rider preference.
type WindsurfProfile struct {
	UserID       string     `json:"-"`
	Level        SkillLevel `json:"level"`
	WindSpeedMin float64    `json:"windSpeedMin"`
	WindSpeedMax float64    `json:"windSpeedMax"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// Spot is a candidate windsurfing location. It can reference a saved location
// (LocationID) or be supplied inline. ShoreNormalDeg is the compass bearing
// pointing from the shore towards the open sea; it is required to classify the
// wind as onshore, offshore or cross-shore.
type Spot struct {
	LocationID     *int64   `json:"locationId,omitempty"`
	Name           string   `json:"name,omitempty"`
	Latitude       *float64 `json:"latitude,omitempty"`
	Longitude      *float64 `json:"longitude,omitempty"`
	ShoreNormalDeg *float64 `json:"shoreNormalDeg,omitempty"`
}

// Orientation classifies wind direction relative to the shoreline.
type Orientation string

// Wind orientation values derived from the spot's shore normal.
const (
	OrientationOnshore    Orientation = "onshore"
	OrientationOffshore   Orientation = "offshore"
	OrientationCrossShore Orientation = "cross-shore"
	OrientationUnknown    Orientation = "unknown"
)

// Recommendation is a concrete spot + time slot combination with a human
// readable justification.
type Recommendation struct {
	IsBest               bool        `json:"isBest"`
	SpotName             string      `json:"spotName"`
	LocationID           *int64      `json:"locationId,omitempty"`
	Latitude             float64     `json:"latitude"`
	Longitude            float64     `json:"longitude"`
	Start                time.Time   `json:"start"`
	End                  time.Time   `json:"end"`
	Score                float64     `json:"score"`
	AverageWindSpeed     float64     `json:"averageWindSpeed"`
	AverageWindDirection float64     `json:"averageWindDirection"`
	AverageWindGusts     float64     `json:"averageWindGusts"`
	Orientation          Orientation `json:"orientation"`
	IsStale              bool        `json:"isStale"`
	Reasons              []string    `json:"reasons"`
}

// RecommendationResult is the payload returned by the windsurf endpoint.
type RecommendationResult struct {
	GeneratedAt     time.Time        `json:"generatedAt"`
	Profile         WindsurfProfile  `json:"profile"`
	Recommendations []Recommendation `json:"recommendations"`
	Message         string           `json:"message,omitempty"`
}
