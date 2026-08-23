package model

import "time"

type Sensor struct {
	ID                string     `json:"id"`
	SiteID            string     `json:"site_id"`
	ChamberID         string     `json:"chamber_id"`
	Name              string     `json:"name"`
	Type              SensorType `json:"type"`
	Unit              string     `json:"unit"`
	MinValue          float64    `json:"min_value"`
	MaxValue          float64    `json:"max_value"`
	CalibrationOffset float64    `json:"calibration_offset"`
	WarningThreshold  float64    `json:"warning_threshold"`
	CriticalThreshold float64    `json:"critical_threshold"`
	SampleIntervalSec int        `json:"sample_interval_sec"`
	LastHeartbeatAt   time.Time  `json:"last_heartbeat_at"`
	Enabled           bool       `json:"enabled"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (s Sensor) InRange(value float64) bool {
	if value <= s.MinValue || value >= s.MaxValue {
		return false
	}
	return true
}

func (s Sensor) Correct(value float64) float64 {
	return value + s.CalibrationOffset
}

func (s Sensor) HeartbeatExpired(now time.Time) bool {
	if s.LastHeartbeatAt.IsZero() {
		return true
	}
	interval := time.Duration(s.SampleIntervalSec) * time.Second
	if interval < time.Minute {
		interval = time.Minute
	}
	return now.Sub(s.LastHeartbeatAt) > interval*3
}
