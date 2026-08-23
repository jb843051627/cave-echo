package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jb843051627/cave-echo/internal/cache"
	"github.com/jb843051627/cave-echo/internal/clock"
	"github.com/jb843051627/cave-echo/internal/engine"
	"github.com/jb843051627/cave-echo/internal/ingest"
	"github.com/jb843051627/cave-echo/internal/metrics"
	"github.com/jb843051627/cave-echo/internal/model"
	"github.com/jb843051627/cave-echo/internal/store"
	"github.com/jb843051627/cave-echo/internal/validation"
)

var (
	ErrNotFound      = store.ErrNotFound
	ErrDuplicate     = errors.New("service: record already exists")
	ErrInvalid       = errors.New("service: invalid input")
	ErrIllegalStage  = errors.New("service: stage transition not allowed")
	ErrSeasonBlocked = errors.New("service: action blocked by seasonal protection rule")
)

type Dependencies struct {
	Store       *store.Store
	Engine      *engine.Engine
	Cache       *cache.Snapshot
	Clock       clock.Clock
	Metrics     *metrics.Registry
	Queue       *ingest.Queue
	DedupWindow time.Duration
	Limits      validation.Limits
}

type Service struct {
	store   *store.Store
	engine  *engine.Engine
	cache   *cache.Snapshot
	clock   clock.Clock
	metrics *metrics.Registry
	queue   *ingest.Queue
	dedup   time.Duration
	limits  validation.Limits
}

func New(deps Dependencies) *Service {
	svc := &Service{
		store:   deps.Store,
		engine:  deps.Engine,
		cache:   deps.Cache,
		clock:   deps.Clock,
		metrics: deps.Metrics,
		queue:   deps.Queue,
		dedup:   deps.DedupWindow,
		limits:  deps.Limits,
	}
	if svc.clock == nil {
		svc.clock = clock.System{}
	}
	if svc.dedup <= 0 {
		svc.dedup = 6 * time.Hour
	}
	if svc.limits.FutureSkew == 0 {
		svc.limits = validation.DefaultLimits()
	}
	return svc
}

func (s *Service) now() time.Time { return s.clock.Now().UTC() }

func (s *Service) bump(metric string) {
	s.metrics.Add(metric, 1)
}

// raiseAlert applies the dedup-window state machine:
// an active alert with the same key is bumped in place; otherwise a new
// open alert is created.
func (s *Service) raiseAlert(proposal engine.ProposedAlert, now time.Time) error {
	dedupKey := strings.Join([]string{string(proposal.Kind), proposal.Rule, proposal.ChamberID, proposal.SensorID}, "|")
	existing, err := s.store.GetActiveAlertByDedupKey(dedupKey)
	switch {
	case err == nil:
		return s.store.BumpAlert(existing.ID, proposal.Severity, proposal.Message, now)
	case errors.Is(err, store.ErrNotFound):
		alert := model.Alert{
			ID:          model.NewID("alr"),
			SiteID:      proposal.SiteID,
			ChamberID:   proposal.ChamberID,
			SensorID:    proposal.SensorID,
			Kind:        proposal.Kind,
			Severity:    proposal.Severity,
			Status:      model.AlertOpen,
			DedupKey:    dedupKey,
			Message:     proposal.Message,
			FirstSeenAt: model.EnsureTime(now),
			LastSeenAt:  model.EnsureTime(now),
			Occurrences: 1,
		}
		if err := s.store.CreateAlert(alert); err != nil {
			if errors.Is(err, store.ErrDuplicateAlert) {
				return nil // raced with another worker; acceptable
			}
			return err
		}
		s.bump("alerts_raised")
		return nil
	default:
		return err
	}
}

// requireSite loads a site or maps the failure into a service error.
func (s *Service) requireSite(siteID string) (model.CaveSite, error) {
	if !model.IsID(siteID) {
		return model.CaveSite{}, fmt.Errorf("%w: site id required", ErrInvalid)
	}
	site, err := s.store.GetSite(siteID)
	if err != nil {
		return model.CaveSite{}, err
	}
	return site, nil
}

func (s *Service) requireChamber(chamberID string) (model.Chamber, error) {
	if !model.IsID(chamberID) {
		return model.Chamber{}, fmt.Errorf("%w: chamber id required", ErrInvalid)
	}
	chamber, err := s.store.GetChamber(chamberID)
	if err != nil {
		return model.Chamber{}, err
	}
	return chamber, nil
}
