package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
	"github.com/jb843051627/cave-echo/internal/store"
	"github.com/jb843051627/cave-echo/internal/validation"
)

// CreateSite registers a new cave site with validation of code, coordinates
// and protection grade.
func (s *Service) CreateSite(input model.CreateSiteInput) (model.CaveSite, error) {
	code := strings.ToUpper(strings.TrimSpace(input.Code))
	if err := validation.RequireText("code", code, 32); err != nil {
		return model.CaveSite{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if err := validation.RequireText("name", input.Name, 120); err != nil {
		return model.CaveSite{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if !input.ProtectionGrade.Valid() {
		return model.CaveSite{}, fmt.Errorf("%w: unknown protection grade %q", ErrInvalid, input.ProtectionGrade)
	}
	if input.Latitude < -90 || input.Latitude > 90 || input.Longitude < -180 || input.Longitude > 180 {
		return model.CaveSite{}, fmt.Errorf("%w: coordinates out of range", ErrInvalid)
	}
	if existing, err := s.store.GetSiteByCode(code); err == nil && existing.ID != "" {
		return model.CaveSite{}, fmt.Errorf("%w: site code %s", ErrDuplicate, code)
	}
	now := s.now()
	site := model.CaveSite{
		ID:              model.NewID("site"),
		Code:            code,
		Name:            strings.TrimSpace(input.Name),
		Latitude:        input.Latitude,
		Longitude:       input.Longitude,
		ElevationM:      input.ElevationM,
		ProtectionGrade: input.ProtectionGrade,
		Status:          model.SiteOpen,
		Description:     strings.TrimSpace(input.Description),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.store.CreateSite(site); err != nil {
		return model.CaveSite{}, mapDuplicate(err, "site")
	}
	s.bump("sites_created")
	return site, nil
}

func (s *Service) GetSite(siteID string) (model.CaveSite, error) {
	return s.requireSite(siteID)
}

func (s *Service) ListSites() ([]model.CaveSite, error) {
	return s.store.ListSites()
}

// UpdateSiteStatus applies the open/restricted/closed lifecycle with a guard:
// a site with critical active alerts cannot be reopened.
func (s *Service) UpdateSiteStatus(siteID string, status model.SiteStatus) (model.CaveSite, error) {
	site, err := s.requireSite(siteID)
	if err != nil {
		return model.CaveSite{}, err
	}
	if !status.Valid() {
		return model.CaveSite{}, fmt.Errorf("%w: unknown status %q", ErrInvalid, status)
	}
	if status == model.SiteOpen && site.Status != model.SiteOpen {
		_, critical, err := s.store.CountActiveAlerts(siteID)
		if err != nil {
			return model.CaveSite{}, err
		}
		if critical > 0 {
			return model.CaveSite{}, fmt.Errorf("%w: resolve %d critical alerts before reopening", ErrInvalid, critical)
		}
	}
	now := s.now()
	if err := s.store.UpdateSiteStatus(siteID, status, now); err != nil {
		return model.CaveSite{}, err
	}
	s.bump("site_status_changes")
	site.Status = status
	site.UpdatedAt = now
	return site, nil
}

// SiteSummary aggregates chambers, sensors, alerts and completeness for one site.
func (s *Service) SiteSummary(siteID string) (model.SiteSummary, error) {
	site, err := s.requireSite(siteID)
	if err != nil {
		return model.SiteSummary{}, err
	}
	chambers, err := s.store.ListChambersBySite(siteID)
	if err != nil {
		return model.SiteSummary{}, err
	}
	sensors, err := s.store.ListSensorsBySite(siteID)
	if err != nil {
		return model.SiteSummary{}, err
	}
	activeAlerts, _, err := s.store.CountActiveAlerts(siteID)
	if err != nil {
		return model.SiteSummary{}, err
	}
	now := s.now()
	windowStart := now.Add(-24 * time.Hour)
	completeness, lastReading, err := s.siteCompleteness(siteID, sensors, windowStart, now)
	if err != nil {
		return model.SiteSummary{}, err
	}
	message := ""
	for _, chamber := range chambers {
		if msg := s.engine.ProtectionMessage(chamber, now); msg != "" {
			message = msg
			break
		}
	}
	return model.SiteSummary{
		Site:              site,
		ChamberCount:      len(chambers),
		SensorCount:       len(sensors),
		ActiveAlertCount:  activeAlerts,
		Completeness:      completeness,
		LastReadingAt:     lastReading,
		ProtectionMessage: message,
	}, nil
}

func (s *Service) siteCompleteness(siteID string, sensors []model.Sensor, from, to time.Time) (float64, time.Time, error) {
	if len(sensors) == 0 {
		return 0, time.Time{}, nil
	}
	counts, err := s.store.CountAcceptedInWindow(siteID, from, to)
	if err != nil {
		return 0, time.Time{}, err
	}
	hours := to.Sub(from).Hours()
	expectedPerSensor := hours * 3600 / expectedInterval(sensors)
	totalExpected := expectedPerSensor * float64(len(sensors))
	totalActual := 0
	var latest time.Time
	for _, sensor := range sensors {
		totalActual += counts[sensor.ID]
		if sensor.LastHeartbeatAt.After(latest) {
			latest = sensor.LastHeartbeatAt
		}
	}
	if totalExpected <= 0 {
		return 0, latest, nil
	}
	ratio := float64(totalActual) / totalExpected
	if ratio > 1 {
		ratio = 1
	}
	return ratio, latest, nil
}

func expectedInterval(sensors []model.Sensor) float64 {
	if len(sensors) == 0 {
		return 300
	}
	sum := 0.0
	for _, sensor := range sensors {
		interval := float64(sensor.SampleIntervalSec)
		if interval <= 0 {
			interval = 300
		}
		sum += interval
	}
	return sum / float64(len(sensors))
}

func mapDuplicate(err error, entity string) error {
	if errors.Is(err, store.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
