package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/jb843051627/cave-echo/internal/engine"
	"github.com/jb843051627/cave-echo/internal/model"
)

// RegisterSensor validates ranges and thresholds, then stores the sensor.
func (s *Service) RegisterSensor(input model.CreateSensorInput) (model.Sensor, error) {
	chamber, err := s.requireChamber(input.ChamberID)
	if err != nil {
		return model.Sensor{}, err
	}
	if !input.Type.Valid() {
		return model.Sensor{}, fmt.Errorf("%w: unknown sensor type %q", ErrInvalid, input.Type)
	}
	if strings.TrimSpace(input.Name) == "" {
		return model.Sensor{}, fmt.Errorf("%w: sensor name required", ErrInvalid)
	}
	if input.MinValue >= input.MaxValue {
		return model.Sensor{}, fmt.Errorf("%w: sensor range inverted", ErrInvalid)
	}
	if input.WarningThreshold != 0 && (input.WarningThreshold < input.MinValue || input.WarningThreshold > input.MaxValue) {
		return model.Sensor{}, fmt.Errorf("%w: warning threshold outside range", ErrInvalid)
	}
	if input.CriticalThreshold != 0 && (input.CriticalThreshold < input.WarningThreshold || input.CriticalThreshold > input.MaxValue) {
		return model.Sensor{}, fmt.Errorf("%w: critical threshold must exceed warning", ErrInvalid)
	}
	if input.SampleIntervalSec <= 0 || input.SampleIntervalSec > 24*3600 {
		input.SampleIntervalSec = 300
	}
	now := s.now()
	sensor := model.Sensor{
		ID:                model.NewID("sen"),
		SiteID:            chamber.SiteID,
		ChamberID:         chamber.ID,
		Name:              strings.TrimSpace(input.Name),
		Type:              input.Type,
		Unit:              defaultUnit(input.Type, input.Unit),
		MinValue:          input.MinValue,
		MaxValue:          input.MaxValue,
		CalibrationOffset: input.CalibrationOffset,
		WarningThreshold:  input.WarningThreshold,
		CriticalThreshold: input.CriticalThreshold,
		SampleIntervalSec: input.SampleIntervalSec,
		Enabled:           true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.store.CreateSensor(sensor); err != nil {
		return model.Sensor{}, err
	}
	s.bump("sensors_registered")
	return sensor, nil
}

func (s *Service) GetSensor(sensorID string) (model.Sensor, error) {
	if !model.IsID(sensorID) {
		return model.Sensor{}, fmt.Errorf("%w: sensor id required", ErrInvalid)
	}
	return s.store.GetSensor(sensorID)
}

func (s *Service) ListSensors(siteID string) ([]model.Sensor, error) {
	if siteID == "" {
		return nil, fmt.Errorf("%w: site id required", ErrInvalid)
	}
	if _, err := s.requireSite(siteID); err != nil {
		return nil, err
	}
	return s.store.ListSensorsBySite(siteID)
}

func (s *Service) SetSensorEnabled(sensorID string, enabled bool) (model.Sensor, error) {
	sensor, err := s.GetSensor(sensorID)
	if err != nil {
		return model.Sensor{}, err
	}
	if err := s.store.SetSensorEnabled(sensorID, enabled, s.now()); err != nil {
		return model.Sensor{}, err
	}
	sensor.Enabled = enabled
	return sensor, nil
}

func (s *Service) UpdateSensorThresholds(sensorID string, warning, critical float64) (model.Sensor, error) {
	sensor, err := s.GetSensor(sensorID)
	if err != nil {
		return model.Sensor{}, err
	}
	if warning != 0 && (warning < sensor.MinValue || warning > sensor.MaxValue) {
		return model.Sensor{}, fmt.Errorf("%w: warning threshold outside range", ErrInvalid)
	}
	if critical != 0 && (critical < warning || critical > sensor.MaxValue) {
		return model.Sensor{}, fmt.Errorf("%w: critical threshold must exceed warning", ErrInvalid)
	}
	if err := s.store.UpdateThresholds(sensorID, warning, critical, s.now()); err != nil {
		return model.Sensor{}, err
	}
	sensor.WarningThreshold = warning
	sensor.CriticalThreshold = critical
	return sensor, nil
}

// EvaluateOfflineSensors scans enabled sensors whose heartbeat expired and
// raises deduplicated offline alerts. Returns how many alerts were touched.
func (s *Service) EvaluateOfflineSensors(maxAge time.Duration) (int, error) {
	sensors, err := s.store.ListEnabledSensors()
	if err != nil {
		return 0, err
	}
	now := s.now()
	touched := 0
	for _, sensor := range sensors {
		reference := sensor.LastHeartbeatAt
		effective := sensor
		effective.LastHeartbeatAt = reference
		if now.Sub(reference) >= maxAge {
			continue
		}
		if err := s.raiseAlert(engine.SensorOfflineAlert(effective, now), now); err != nil {
			return touched, err
		}
		touched++
	}
	return touched, nil
}

func defaultUnit(kind model.SensorType, unit string) string {
	if strings.TrimSpace(unit) != "" {
		return strings.TrimSpace(unit)
	}
	switch kind {
	case model.SensorTemperature:
		return "°C"
	case model.SensorHumidity:
		return "%RH"
	case model.SensorCO2:
		return "ppm"
	case model.SensorPressure:
		return "hPa"
	default:
		return "raw"
	}
}
