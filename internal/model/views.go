package model

import "time"

type ChamberSnapshot struct {
	Chamber        Chamber              `json:"chamber"`
	Assessment     *StabilityAssessment `json:"assessment,omitempty"`
	LatestReadings []Reading            `json:"latest_readings"`
	LatestGas      *GasSample           `json:"latest_gas,omitempty"`
	DripTrend      DripTrend            `json:"drip_trend"`
	ActiveAlerts   int                  `json:"active_alerts"`
}

type SiteSummary struct {
	Site              CaveSite  `json:"site"`
	ChamberCount      int       `json:"chamber_count"`
	SensorCount       int       `json:"sensor_count"`
	ActiveAlertCount  int       `json:"active_alert_count"`
	Completeness      float64   `json:"completeness"`
	LastReadingAt     time.Time `json:"last_reading_at"`
	ProtectionMessage string    `json:"protection_message"`
}

type Overview struct {
	GeneratedAt     time.Time     `json:"generated_at"`
	Sites           []SiteSummary `json:"sites"`
	OpenAlerts      int           `json:"open_alerts"`
	CriticalAlerts  int           `json:"critical_alerts"`
	ReadingsToday   int           `json:"readings_today"`
	Observability   float64       `json:"observability"`
	SeasonalNotices []string      `json:"seasonal_notices"`
}

type ExportBundle struct {
	GeneratedAt time.Time             `json:"generated_at"`
	Sites       []CaveSite            `json:"sites"`
	Chambers    []Chamber             `json:"chambers"`
	Sensors     []Sensor              `json:"sensors"`
	Readings    []Reading             `json:"readings"`
	Drips       []DripEvent           `json:"drips"`
	GasSamples  []GasSample           `json:"gas_samples"`
	Surveys     []Survey              `json:"surveys"`
	Assessments []StabilityAssessment `json:"assessments"`
	Alerts      []Alert               `json:"alerts"`
	Notes       []ConservationNote    `json:"notes"`
}
