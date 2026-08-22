package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
)

// CreateChamber adds a chamber partition under an existing site.
func (s *Service) CreateChamber(siteID string, input model.CreateChamberInput) (model.Chamber, error) {
	site, err := s.requireSite(siteID)
	if err != nil {
		return model.Chamber{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return model.Chamber{}, fmt.Errorf("%w: chamber name required", ErrInvalid)
	}
	band := strings.TrimSpace(input.TemperatureBand)
	if band != "" && !validTemperatureBand(band) {
		return model.Chamber{}, fmt.Errorf("%w: unknown temperature band %q", ErrInvalid, band)
	}
	now := s.now()
	chamber := model.Chamber{
		ID:                model.NewID("chm"),
		SiteID:            site.ID,
		Name:              name,
		TemperatureBand:   band,
		AirflowDirection:  strings.TrimSpace(input.AirflowDirection),
		IsolationBoundary: strings.TrimSpace(input.IsolationBoundary),
		BatHabitat:        input.BatHabitat,
		ProtectionRule:    strings.TrimSpace(input.ProtectionRule),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.store.CreateChamber(chamber); err != nil {
		return model.Chamber{}, err
	}
	s.bump("chambers_created")
	return chamber, nil
}

func (s *Service) ListChambers(siteID string) ([]model.Chamber, error) {
	if _, err := s.requireSite(siteID); err != nil {
		return nil, err
	}
	return s.store.ListChambersBySite(siteID)
}

func (s *Service) ListAllChambers() ([]model.Chamber, error) {
	return s.store.ListAllChambers()
}

// ChamberSnapshot assembles the live view used by the console: latest
// readings per sensor type from the cache (falling back to the store), the
// most recent gas sample, the drip trend and the newest assessment.
func (s *Service) ChamberSnapshot(chamberID string) (model.ChamberSnapshot, error) {
	chamber, err := s.requireChamber(chamberID)
	if err != nil {
		return model.ChamberSnapshot{}, err
	}
	now := s.now()
	readings := s.cache.ChamberReadings(chamber.SiteID, chamber.ID)
	if len(readings) == 0 {
		stored, storeErr := s.store.LatestReadingPerType(chamber.ID)
		if storeErr != nil {
			return model.ChamberSnapshot{}, storeErr
		}
		readings = stored
	}
	gasSample, err := s.store.LatestGasSample(chamber.ID)
	hasGas := true
	if err != nil {
		if err != ErrNotFound {
			return model.ChamberSnapshot{}, err
		}
		hasGas = false
	}
	var gasPtr *model.GasSample
	if hasGas {
		gasPtr = &gasSample
	}
	events, err := s.store.ListDripEvents(chamber.ID, now.Add(-30*24*time.Hour), now, 500)
	if err != nil {
		return model.ChamberSnapshot{}, err
	}
	trend := s.engine.DripTrend(events)
	assessment, err := s.store.LatestAssessment(chamber.ID)
	var assessmentPtr *model.StabilityAssessment
	if err == nil {
		assessmentPtr = &assessment
	} else if err != ErrNotFound {
		return model.ChamberSnapshot{}, err
	}
	alerts, err := s.store.ListAlerts("", chamber.ID, "active", 1000)
	if err != nil {
		return model.ChamberSnapshot{}, err
	}
	return model.ChamberSnapshot{
		Chamber:        chamber,
		Assessment:     assessmentPtr,
		LatestReadings: readings,
		LatestGas:      gasPtr,
		DripTrend:      trend,
		ActiveAlerts:   len(alerts),
	}, nil
}

func validTemperatureBand(band string) bool {
	switch band {
	case "cold", "cool", "temperate", "warm", "variable":
		return true
	default:
		return false
	}
}
