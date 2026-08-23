package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/jb843051627/cave-echo/internal/engine"
	"github.com/jb843051627/cave-echo/internal/model"
	"github.com/jb843051627/cave-echo/internal/validation"
)

// RecordGasSample validates a gas sample, stores it and raises gas alerts
// when CO2/O2 cross the safety bounds.
func (s *Service) RecordGasSample(input model.CreateGasInput) (model.GasSample, error) {
	chamber, err := s.requireChamber(input.ChamberID)
	if err != nil {
		return model.GasSample{}, err
	}
	if !input.Method.Valid() {
		return model.GasSample{}, fmt.Errorf("%w: unknown sampling method %q", ErrInvalid, input.Method)
	}
	if input.CO2PPM < 0 || input.CO2PPM > 200000 {
		return model.GasSample{}, fmt.Errorf("%w: co2 value implausible", ErrInvalid)
	}
	if input.OxygenPercent < 0 || input.OxygenPercent > 100 {
		return model.GasSample{}, fmt.Errorf("%w: oxygen percent out of range", ErrInvalid)
	}
	now := s.now()
	if input.SampledAt.IsZero() {
		input.SampledAt = now
	}
	if err := validation.ValidateTimestamp(input.SampledAt, s.limits, now); err != nil {
		return model.GasSample{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	sample := model.GasSample{
		ID:              model.NewID("gas"),
		ChamberID:       chamber.ID,
		SampledAt:       model.EnsureTime(input.SampledAt),
		CO2PPM:          input.CO2PPM,
		OxygenPercent:   input.OxygenPercent,
		RadonBqM3:       input.RadonBqM3,
		TemperatureC:    input.TemperatureC,
		HumidityPercent: input.HumidityPercent,
		Method:          input.Method,
		Technician:      strings.TrimSpace(input.Technician),
		CreatedAt:       now,
	}
	if err := s.store.CreateGasSample(sample); err != nil {
		return model.GasSample{}, err
	}
	s.bump("gas_samples_recorded")

	risk := s.engine.GasRisk(sample)
	switch {
	case risk >= 0.8:
		err = s.raiseAlert(engine.ProposedAlert{
			SiteID:    chamber.SiteID,
			ChamberID: chamber.ID,
			Kind:      model.AlertGas,
			Severity:  model.SeverityCritical,
			Message: fmt.Sprintf("critical gas mix in %s: co2=%.0fppm o2=%.1f%% risk=%.2f",
				chamber.Name, sample.CO2PPM, sample.OxygenPercent, risk),
			Rule: "sample_gas_critical",
		}, now)
	case risk >= 0.5:
		err = s.raiseAlert(engine.ProposedAlert{
			SiteID:    chamber.SiteID,
			ChamberID: chamber.ID,
			Kind:      model.AlertGas,
			Severity:  model.SeverityWarning,
			Message: fmt.Sprintf("elevated gas reading in %s: co2=%.0fppm o2=%.1f%% risk=%.2f",
				chamber.Name, sample.CO2PPM, sample.OxygenPercent, risk),
			Rule: "sample_gas_elevated",
		}, now)
	}
	if err != nil {
		return sample, err
	}
	return sample, nil
}

func (s *Service) ListGasSamples(chamberID string, limit int) ([]model.GasSample, error) {
	if _, err := s.requireChamber(chamberID); err != nil {
		return nil, err
	}
	return s.store.ListGasSamples(chamberID, s.now().Add(-365*24*time.Hour), s.now(), limit)
}

func (s *Service) LatestGasSample(chamberID string) (model.GasSample, error) {
	if _, err := s.requireChamber(chamberID); err != nil {
		return model.GasSample{}, err
	}
	return s.store.LatestGasSample(chamberID)
}
