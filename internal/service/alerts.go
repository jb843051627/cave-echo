package service

import (
	"errors"
	"fmt"

	"github.com/jb843051627/cave-echo/internal/model"
	"github.com/jb843051627/cave-echo/internal/store"
)

// ListAlerts filters alerts by site/chamber/status.
func (s *Service) ListAlerts(siteID, chamberID, status string, limit int) ([]model.Alert, error) {
	if siteID != "" {
		if _, err := s.requireSite(siteID); err != nil {
			return nil, err
		}
	}
	return s.store.ListAlerts(siteID, chamberID, status, limit)
}

// AcknowledgeAlert moves open → acknowledged; already acked alerts are idempotent.
func (s *Service) AcknowledgeAlert(alertID string) (model.Alert, error) {
	alert, err := s.loadAlert(alertID)
	if err != nil {
		return model.Alert{}, err
	}
	if alert.Status == model.AlertAcknowledged {
		return alert, nil
	}
	if !alert.CanAcknowledge() {
		return model.Alert{}, fmt.Errorf("%w: alert %s is %s", ErrInvalid, alert.ID, alert.Status)
	}
	now := s.now()
	if err := s.store.AcknowledgeAlert(alertID, now); err != nil {
		return model.Alert{}, err
	}
	s.bump("alerts_acked")
	alert.Status = model.AlertAcknowledged
	alert.AcknowledgedAt = now
	return alert, nil
}

// CloseAlert applies the acked→resolved state machine: critical alerts must
// be acknowledged before they can be closed.
func (s *Service) CloseAlert(alertID string) (model.Alert, error) {
	alert, err := s.loadAlert(alertID)
	if err != nil {
		return model.Alert{}, err
	}
	if alert.Status == model.AlertClosed {
		return alert, nil
	}
	if !alert.CanClose() {
		return model.Alert{}, fmt.Errorf("%w: critical alert must be acknowledged before closing", ErrInvalid)
	}
	now := s.now()
	if err := s.store.CloseAlert(alertID, now); err != nil {
		return model.Alert{}, err
	}
	s.bump("alerts_closed")
	alert.Status = model.AlertClosed
	alert.ClosedAt = now
	return alert, nil
}

// ExpireStaleAlerts closes alerts that have not recurred within the dedup window.
func (s *Service) ExpireStaleAlerts() (int64, error) {
	closed, err := s.store.ExpireStaleAlerts(s.dedup, s.now())
	if closed > 0 {
		s.metrics.Add("alerts_expired", closed)
	}
	return closed, err
}

func (s *Service) loadAlert(alertID string) (model.Alert, error) {
	if !model.IsID(alertID) {
		return model.Alert{}, fmt.Errorf("%w: alert id required", ErrInvalid)
	}
	alert, err := s.store.GetAlert(alertID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return model.Alert{}, ErrNotFound
		}
		return model.Alert{}, err
	}
	return alert, nil
}
