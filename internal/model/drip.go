package model

import "time"

type DripEvent struct {
	ID              string    `json:"id"`
	ChamberID       string    `json:"chamber_id"`
	ObservedAt      time.Time `json:"observed_at"`
	RatePerMinute   float64   `json:"rate_per_minute"`
	Mineralization  float64   `json:"mineralization_mg_l"`
	Color           DripColor `json:"color"`
	Location        string    `json:"location"`
	DurationSeconds int       `json:"duration_seconds"`
	Observer        string    `json:"observer"`
	CreatedAt       time.Time `json:"created_at"`
}

type DripTrend struct {
	AverageRate     float64   `json:"average_rate"`
	RecentRate      float64   `json:"recent_rate"`
	SlopePerDay     float64   `json:"slope_per_day"`
	EventCount      int       `json:"event_count"`
	ObservationFrom time.Time `json:"observation_from"`
	ObservationTo   time.Time `json:"observation_to"`
	Direction       string    `json:"direction"`
}
