package model

import "time"

type CaveSite struct {
	ID              string          `json:"id"`
	Code            string          `json:"code"`
	Name            string          `json:"name"`
	Latitude        float64         `json:"latitude"`
	Longitude       float64         `json:"longitude"`
	ElevationM      float64         `json:"elevation_m"`
	ProtectionGrade ProtectionGrade `json:"protection_grade"`
	Status          SiteStatus      `json:"status"`
	Description     string          `json:"description"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func (s CaveSite) IsAccessible() bool {
	return s.Status == SiteOpen
}

func (s CaveSite) CoordinatesValid() bool {
	return s.Latitude >= -90 && s.Latitude <= 90 && s.Longitude >= -180 && s.Longitude <= 180
}

func (s CaveSite) Touch(now time.Time) CaveSite {
	s.UpdatedAt = EnsureTime(now)
	return s
}
