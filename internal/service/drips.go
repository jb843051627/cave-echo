package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/jb843051627/cave-echo/internal/engine"
	"github.com/jb843051627/cave-echo/internal/model"
	"github.com/jb843051627/cave-echo/internal/validation"
)

// RecordDripEvent validates and stores a drip observation, then re-scores the
// chamber drip regime and raises an alert when the trend is anomalous.
func (s *Service) RecordDripEvent(input model.CreateDripInput) (model.DripEvent, error) {
	chamber, err := s.requireChamber(input.ChamberID)
	if err != nil {
		return model.DripEvent{}, err
	}
	if !input.Color.Valid() {
		return model.DripEvent{}, fmt.Errorf("%w: unknown drip color %q", ErrInvalid, input.Color)
	}
	if input.RatePerMinute < 0 || input.RatePerMinute > 1000 {
		return model.DripEvent{}, fmt.Errorf("%w: drip rate out of plausible range", ErrInvalid)
	}
	now := s.now()
	if input.ObservedAt.IsZero() {
		input.ObservedAt = now
	}
	if err := validation.ValidateTimestamp(input.ObservedAt, s.limits, now); err != nil {
		return model.DripEvent{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	event := model.DripEvent{
		ID:              model.NewID("drp"),
		ChamberID:       chamber.ID,
		ObservedAt:      model.EnsureTime(input.ObservedAt),
		RatePerMinute:   input.RatePerMinute,
		Mineralization:  input.Mineralization,
		Color:           input.Color,
		Location:        strings.TrimSpace(input.Location),
		DurationSeconds: input.DurationSeconds,
		Observer:        strings.TrimSpace(input.Observer),
		CreatedAt:       now,
	}
	if err := s.store.CreateDripEvent(event); err != nil {
		return model.DripEvent{}, err
	}
	s.bump("drip_events_recorded")

	trend, events, err := s.chamberDripContext(chamber.ID, now)
	if err != nil {
		return event, err
	}
	risk := s.engine.DripRisk(trend, events)
	if risk >= 0.7 {
		err := s.raiseAlert(engine.ProposedAlert{
			SiteID:    chamber.SiteID,
			ChamberID: chamber.ID,
			Kind:      model.AlertDrip,
			Severity:  model.SeverityWarning,
			Message: fmt.Sprintf("drip anomaly at %s: direction=%s slope=%.2f/day risk=%.2f",
				chamber.Name, trend.Direction, trend.SlopePerDay, risk),
			Rule: "drip_trend",
		}, now)
		if err != nil {
			return event, err
		}
	}
	return event, nil
}

func (s *Service) ListDripEvents(chamberID string, from, to time.Time, limit int) ([]model.DripEvent, error) {
	if _, err := s.requireChamber(chamberID); err != nil {
		return nil, err
	}
	return s.store.ListDripEvents(chamberID, from, to, limit)
}

func (s *Service) DripTrend(chamberID string, window time.Duration) (model.DripTrend, error) {
	_, events, err := s.dripWindow(chamberID, window)
	if err != nil {
		return model.DripTrend{}, err
	}
	return s.engine.DripTrend(events), nil
}

func (s *Service) dripWindow(chamberID string, window time.Duration) (time.Time, []model.DripEvent, error) {
	if _, err := s.requireChamber(chamberID); err != nil {
		return time.Time{}, nil, err
	}
	if window <= 0 {
		window = 30 * 24 * time.Hour
	}
	to := s.now()
	events, err := s.store.ListDripEvents(chamberID, to.Add(-window), to, 2000)
	if err != nil {
		return time.Time{}, nil, err
	}
	return to, events, nil
}

func (s *Service) chamberDripContext(chamberID string, now time.Time) (model.DripTrend, []model.DripEvent, error) {
	events, err := s.store.ListDripEvents(chamberID, now.Add(-30*24*time.Hour), now, 2000)
	if err != nil {
		return model.DripTrend{}, nil, err
	}
	return s.engine.DripTrend(events), events, nil
}
