package service

import (
	"fmt"
	"time"

	"github.com/jb843051627/cave-echo/internal/engine"
	"github.com/jb843051627/cave-echo/internal/model"
)

// RunAssessment evaluates one chamber: it gathers the latest readings, the
// newest gas sample and the drip history, computes all risk dimensions and
// persists a StabilityAssessment. Alerts derived from the assessment go
// through the dedup-window state machine.
func (s *Service) RunAssessment(chamberID string) (model.StabilityAssessment, error) {
	chamber, err := s.requireChamber(chamberID)
	if err != nil {
		return model.StabilityAssessment{}, err
	}
	input, err := s.assessmentInput(chamber)
	if err != nil {
		return model.StabilityAssessment{}, err
	}
	assessment := s.engine.Assess(input)
	assessment.ID = model.NewID("asm")
	assessment.CreatedAt = s.now()
	if err := s.store.CreateAssessment(assessment); err != nil {
		return model.StabilityAssessment{}, err
	}
	s.bump("assessments_run")

	for _, proposal := range s.engine.AssessmentAlerts(assessment) {
		if proposal.SiteID == "" {
			proposal.SiteID = chamber.SiteID
		}
		if err := s.raiseAlert(proposal, s.now()); err != nil {
			return assessment, err
		}
	}

	// Cross the level into site status policy: restricted or closed chambers
	// on an open site raise a protection alert once per dedup window.
	if assessment.RequiresRestriction() {
		site, siteErr := s.store.GetSite(chamber.SiteID)
		if siteErr == nil && site.Status == model.SiteOpen {
			err := s.raiseAlert(engine.ProposedAlert{
				SiteID:    chamber.SiteID,
				ChamberID: chamber.ID,
				Kind:      model.AlertProtection,
				Severity:  model.SeverityWarning,
				Message: fmt.Sprintf("chamber %s assessed %s while site %s is open",
					chamber.Name, assessment.Level, site.Code),
				Rule: "site_open_restricted_chamber",
			}, s.now())
			if err != nil {
				return assessment, err
			}
		}
	}
	return assessment, nil
}

func (s *Service) assessmentInput(chamber model.Chamber) (engine.ChamberInput, error) {
	now := s.now()
	readings := s.cache.ChamberReadings(chamber.SiteID, chamber.ID)
	if len(readings) == 0 {
		stored, err := s.store.LatestReadingPerType(chamber.ID)
		if err != nil {
			return engine.ChamberInput{}, err
		}
		readings = stored
	}
	gasSample, err := s.store.LatestGasSample(chamber.ID)
	var gasPtr *model.GasSample
	switch {
	case err == nil:
		gasPtr = &gasSample
	case err != ErrNotFound:
		return engine.ChamberInput{}, err
	}
	events, err := s.store.ListDripEvents(chamber.ID, now.Add(-30*24*time.Hour), now, 2000)
	if err != nil {
		return engine.ChamberInput{}, err
	}
	trend := s.engine.DripTrend(events)

	sensors, err := s.store.ListSensorsByChamber(chamber.ID)
	if err != nil {
		return engine.ChamberInput{}, err
	}
	expectedRate, actualRate := telemetryRates(sensors, readings, now)
	return engine.ChamberInput{
		Chamber:      chamber,
		Readings:     readings,
		GasSample:    gasPtr,
		DripTrend:    trend,
		DripEvents:   events,
		ExpectedRate: expectedRate,
		ActualRate:   actualRate,
		Now:          now,
	}, nil
}

// telemetryRates converts sensor sample intervals into an hourly expectation
// and derives an observed completeness from heartbeat freshness.
func telemetryRates(sensors []model.Sensor, readings []model.Reading, now time.Time) (float64, float64) {
	if len(sensors) == 0 {
		return 0, 0
	}
	expected := 0.0
	for _, sensor := range sensors {
		interval := time.Duration(sensor.SampleIntervalSec) * time.Second
		if interval <= 0 {
			interval = 5 * time.Minute
		}
		expected += time.Hour.Seconds() / interval.Seconds()
	}
	return expected, completenessFromSensors(sensors, now)
}

func completenessFromSensors(sensors []model.Sensor, now time.Time) float64 {
	if len(sensors) == 0 {
		return 0
	}
	fresh := 0
	for _, sensor := range sensors {
		maxAge := time.Duration(sensor.SampleIntervalSec*3) * time.Second
		if maxAge < 30*time.Minute {
			maxAge = 30 * time.Minute
		}
		if !sensor.LastHeartbeatAt.IsZero() && now.Sub(sensor.LastHeartbeatAt) <= maxAge {
			fresh++
		}
	}
	return float64(fresh) / float64(len(sensors))
}

func (s *Service) ListAssessments(chamberID string, limit int) ([]model.StabilityAssessment, error) {
	if _, err := s.requireChamber(chamberID); err != nil {
		return nil, err
	}
	return s.store.ListAssessments(chamberID, "", limit)
}
