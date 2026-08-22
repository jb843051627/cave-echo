package model

import "time"

type Survey struct {
	ID               string      `json:"id"`
	SiteID           string      `json:"site_id"`
	ChamberID        string      `json:"chamber_id"`
	Transect         string      `json:"transect"`
	SurfaceCondition string      `json:"surface_condition"`
	CrystalChangeMM  float64     `json:"crystal_change_mm"`
	Stage            SurveyStage `json:"stage"`
	StartedAt        time.Time   `json:"started_at"`
	CompletedAt      time.Time   `json:"completed_at"`
	Findings         string      `json:"findings"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

func (s Survey) Active() bool {
	return s.Stage != SurveyClosed
}

func (s Survey) CompletionReady() bool {
	return s.Findings != "" && s.SurfaceCondition != ""
}
