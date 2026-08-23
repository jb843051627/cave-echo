package model

import "time"

type GasSample struct {
	ID              string    `json:"id"`
	ChamberID       string    `json:"chamber_id"`
	SampledAt       time.Time `json:"sampled_at"`
	CO2PPM          float64   `json:"co2_ppm"`
	OxygenPercent   float64   `json:"oxygen_percent"`
	RadonBqM3       float64   `json:"radon_bq_m3"`
	TemperatureC    float64   `json:"temperature_c"`
	HumidityPercent float64   `json:"humidity_percent"`
	Method          GasMethod `json:"method"`
	Technician      string    `json:"technician"`
	CreatedAt       time.Time `json:"created_at"`
}

func (s GasSample) OxygenDeficient() bool {
	return s.OxygenPercent < 19.5
}

func (s GasSample) CO2Risk() string {
	switch {
	case s.CO2PPM >= 5000:
		return "critical"
	case s.CO2PPM >= 2500:
		return "warning"
	default:
		return "normal"
	}
}
