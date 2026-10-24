package domain

import "time"

// Location is a geocoded point on the map. The same physical point is shared
// between users, so it is stored once and referenced by SavedLocation.
type Location struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	CountryCode string    `json:"countryCode"`
	Timezone    string    `json:"timezone,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Coordinate returns the (lat, lon) pair used to identify a unique point.
func (l Location) Coordinate() (float64, float64) {
	return l.Latitude, l.Longitude
}

// SavedLocation links a User to a Location and carries per-user metadata such
// as the display order in the comparison list and an optional windsurfing
// shore orientation (compass bearing pointing from land towards the sea).
type SavedLocation struct {
	LocationID     int64     `json:"locationId"`
	Name           string    `json:"name"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	CountryCode    string    `json:"countryCode"`
	Timezone       string    `json:"timezone,omitempty"`
	Alias          string    `json:"alias,omitempty"`
	DisplayOrder   int       `json:"displayOrder"`
	ShoreNormalDeg *float64  `json:"shoreNormalDeg,omitempty"`
	SavedAt        time.Time `json:"savedAt"`
}
