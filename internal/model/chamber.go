package model

import "time"

type Chamber struct {
	ID                string    `json:"id"`
	SiteID            string    `json:"site_id"`
	Name              string    `json:"name"`
	TemperatureBand   string    `json:"temperature_band"`
	AirflowDirection  string    `json:"airflow_direction"`
	IsolationBoundary string    `json:"isolation_boundary"`
	BatHabitat        bool      `json:"bat_habitat"`
	ProtectionRule    string    `json:"protection_rule"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (c Chamber) NeedsSeasonalReview(month time.Month) bool {
	if !c.BatHabitat {
		return false
	}
	return month >= time.August && month <= time.November
}

func (c Chamber) BoundaryRequired() bool {
	return c.IsolationBoundary != ""
}
