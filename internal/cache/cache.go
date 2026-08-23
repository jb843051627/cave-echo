package cache

import (
	"sync"
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
)

type entry struct {
	readings  map[model.SensorType]model.Reading
	updatedAt time.Time
}

type Snapshot struct {
	mu      sync.RWMutex
	bySite  map[string]map[string]*entry
	latest  map[string]time.Time
	closing bool
}

func New() *Snapshot {
	return &Snapshot{
		bySite: make(map[string]map[string]*entry),
		latest: make(map[string]time.Time),
	}
}

func (s *Snapshot) Apply(reading model.Reading) {
	if reading.SiteID == "" || reading.SensorID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing {
		return
	}
	chambers, ok := s.bySite[reading.SiteID]
	if !ok {
		chambers = make(map[string]*entry)
		s.bySite[reading.SiteID] = chambers
	}
	current, ok := chambers[reading.ChamberID]
	if !ok {
		current = &entry{readings: make(map[model.SensorType]model.Reading)}
		chambers[reading.ChamberID] = current
	}
	existing, ok := current.readings[reading.SensorType]
	if !ok || reading.ObservedAt.After(existing.ObservedAt) {
		current.readings[reading.SensorType] = reading
		current.updatedAt = time.Now().UTC()
	}
}

func (s *Snapshot) ChamberReadings(siteID, chamberID string) []model.Reading {
	s.mu.RLock()
	defer s.mu.RUnlock()
	chambers, ok := s.bySite[siteID]
	if !ok {
		return nil
	}
	current, ok := chambers[chamberID]
	if !ok {
		return nil
	}
	out := make([]model.Reading, 0, len(current.readings))
	for _, reading := range current.readings {
		out = append(out, reading)
	}
	return out
}

func (s *Snapshot) SiteReadings(siteID string) []model.Reading {
	s.mu.RLock()
	defer s.mu.RUnlock()
	chambers, ok := s.bySite[siteID]
	if !ok {
		return nil
	}
	var out []model.Reading
	for _, chamber := range chambers {
		for _, reading := range chamber.readings {
			out = append(out, reading)
		}
	}
	return out
}

func (s *Snapshot) SensorLastSeen(sensorID string) (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.latest[sensorID]
	return value, ok
}

func (s *Snapshot) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bySite = make(map[string]map[string]*entry)
	s.latest = make(map[string]time.Time)
}
