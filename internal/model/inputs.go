package model

import "time"

type CreateSiteInput struct {
	Code            string          `json:"code"`
	Name            string          `json:"name"`
	Latitude        float64         `json:"latitude"`
	Longitude       float64         `json:"longitude"`
	ElevationM      float64         `json:"elevation_m"`
	ProtectionGrade ProtectionGrade `json:"protection_grade"`
	Description     string          `json:"description"`
}

type UpdateSiteStatusInput struct {
	Status SiteStatus `json:"status"`
}

type CreateChamberInput struct {
	Name              string `json:"name"`
	TemperatureBand   string `json:"temperature_band"`
	AirflowDirection  string `json:"airflow_direction"`
	IsolationBoundary string `json:"isolation_boundary"`
	BatHabitat        bool   `json:"bat_habitat"`
	ProtectionRule    string `json:"protection_rule"`
}

type CreateSensorInput struct {
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
}

type TelemetryPoint struct {
	SensorID   string    `json:"sensor_id"`
	ObservedAt time.Time `json:"observed_at"`
	Value      float64   `json:"value"`
}

type TelemetryBatch struct {
	BatchID  string           `json:"batch_id"`
	SiteID   string           `json:"site_id"`
	SentAt   time.Time        `json:"sent_at"`
	Checksum uint32           `json:"checksum"`
	Readings []TelemetryPoint `json:"readings"`
}

type CreateDripInput struct {
	ChamberID       string    `json:"chamber_id"`
	ObservedAt      time.Time `json:"observed_at"`
	RatePerMinute   float64   `json:"rate_per_minute"`
	Mineralization  float64   `json:"mineralization_mg_l"`
	Color           DripColor `json:"color"`
	Location        string    `json:"location"`
	DurationSeconds int       `json:"duration_seconds"`
	Observer        string    `json:"observer"`
}

type CreateGasInput struct {
	ChamberID       string    `json:"chamber_id"`
	SampledAt       time.Time `json:"sampled_at"`
	CO2PPM          float64   `json:"co2_ppm"`
	OxygenPercent   float64   `json:"oxygen_percent"`
	RadonBqM3       float64   `json:"radon_bq_m3"`
	TemperatureC    float64   `json:"temperature_c"`
	HumidityPercent float64   `json:"humidity_percent"`
	Method          GasMethod `json:"method"`
	Technician      string    `json:"technician"`
}

type CreateSurveyInput struct {
	SiteID           string  `json:"site_id"`
	ChamberID        string  `json:"chamber_id"`
	Transect         string  `json:"transect"`
	SurfaceCondition string  `json:"surface_condition"`
	CrystalChangeMM  float64 `json:"crystal_change_mm"`
	Findings         string  `json:"findings"`
}

type TransitionSurveyInput struct {
	Stage    SurveyStage `json:"stage"`
	Findings string      `json:"findings"`
}

type CreateNoteInput struct {
	SiteID        string       `json:"site_id"`
	ChamberID     string       `json:"chamber_id"`
	SurveyID      string       `json:"survey_id"`
	Author        string       `json:"author"`
	Category      NoteCategory `json:"category"`
	Note          string       `json:"note"`
	ActionOutcome string       `json:"action_outcome"`
}
