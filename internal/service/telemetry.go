package service

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jb843051627/cave-echo/internal/engine"
	"github.com/jb843051627/cave-echo/internal/model"
	"github.com/jb843051627/cave-echo/internal/store"
	"github.com/jb843051627/cave-echo/internal/validation"
)

// IngestBatch validates a telemetry batch (time window, ranges, batch
// checksum), persists accepted readings, refreshes the snapshot cache and
// raises threshold alerts. It returns the number of stored readings.
func (s *Service) IngestBatch(batch model.TelemetryBatch) (int, error) {
	now := s.now()
	if err := validation.ValidateBatch(batch, s.limits, now); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if !verifyChecksum(batch) {
		return 0, fmt.Errorf("%w: batch checksum mismatch", ErrInvalid)
	}

	sensorCache := make(map[string]model.Sensor, len(batch.Readings))
	readings := make([]model.Reading, 0, len(batch.Readings))
	for _, point := range batch.Readings {
		sensor, ok := sensorCache[point.SensorID]
		if !ok {
			loaded, err := s.store.GetSensor(point.SensorID)
			if err != nil {
				return 0, fmt.Errorf("%w: sensor %s: %v", ErrInvalid, point.SensorID, err)
			}
			if !loaded.Enabled {
				return 0, fmt.Errorf("%w: sensor %s not part of site or disabled", ErrInvalid, point.SensorID)
			}
			sensorCache[point.SensorID] = loaded
			sensor = loaded
		}
		quality, corrected := validation.ClassifyQuality(sensor, point.Value)
		readings = append(readings, model.Reading{
			ID:         model.NewID("rdg"),
			SensorID:   sensor.ID,
			SiteID:     sensor.SiteID,
			ChamberID:  sensor.ChamberID,
			SensorType: sensor.Type,
			ObservedAt: model.EnsureTime(point.ObservedAt),
			RawValue:   point.Value,
			Value:      corrected,
			Unit:       sensor.Unit,
			Quality:    quality,
			BatchID:    strings.TrimSpace(batch.BatchID),
			Checksum:   batch.Checksum,
			ReceivedAt: now,
		})
	}

	inserted, err := s.store.InsertReadings(readings)
	if err != nil {
		return 0, err
	}
	for _, reading := range readings {
		if reading.Quality == model.QualityRejected {
			continue
		}
		s.cache.Apply(reading)
		if err := s.store.TouchHeartbeat(reading.SensorID, reading.ObservedAt); err != nil {
			return inserted, err
		}
		sensor := sensorCache[reading.SensorID]
		if severity := s.engine.ThresholdBreach(sensor, reading.Value); severity != "" {
			err := s.raiseAlert(engine.ProposedAlert{
				SiteID:    sensor.SiteID,
				ChamberID: sensor.ChamberID,
				SensorID:  sensor.ID,
				Kind:      model.AlertMicroclimate,
				Severity:  severity,
				Message: fmt.Sprintf("sensor %s (%s) value %.2f crossed %s threshold",
					sensor.Name, sensor.Type, reading.Value, severity),
				Rule: "threshold_" + string(sensor.Type),
			}, now)
			if err != nil {
				return inserted, err
			}
		}
	}
	s.bump("readings_ingested")
	return inserted, nil
}

// verifyChecksum recomputes the canonical CRC32 over the ordered points and
// compares it with the declared batch checksum.
func verifyChecksum(batch model.TelemetryBatch) bool {
	points := make([]model.TelemetryPoint, len(batch.Readings))
	copy(points, batch.Readings)
	sort.Slice(points, func(i, j int) bool {
		if points[i].SensorID != points[j].SensorID {
			return points[i].SensorID < points[j].SensorID
		}
		return points[i].ObservedAt.Before(points[j].ObservedAt)
	})
	var payload []byte
	for _, point := range points {
		payload = append(payload, point.SensorID...)
		var stamp [8]byte
		binary.BigEndian.PutUint64(stamp[:], uint64(point.ObservedAt.UnixMilli()))
		payload = append(payload, stamp[:]...)
		var raw [8]byte
		binary.BigEndian.PutUint64(raw[:], math.Float64bits(point.Value))
		payload = append(payload, raw[:]...)
	}
	return validation.Matches(payload, batch.Checksum)
}

// ExpectedChecksum lets clients compute the checksum the same way the server does.
func ExpectedChecksum(points []model.TelemetryPoint) uint32 {
	return computeChecksum(points)
}

func computeChecksum(points []model.TelemetryPoint) uint32 {
	ordered := make([]model.TelemetryPoint, len(points))
	copy(ordered, points)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].SensorID != ordered[j].SensorID {
			return ordered[i].SensorID < ordered[j].SensorID
		}
		return ordered[i].ObservedAt.Before(ordered[j].ObservedAt)
	})
	var payload []byte
	for _, point := range ordered {
		payload = append(payload, point.SensorID...)
		var stamp [8]byte
		binary.BigEndian.PutUint64(stamp[:], uint64(point.ObservedAt.UnixMilli()))
		payload = append(payload, stamp[:]...)
		var raw [8]byte
		binary.BigEndian.PutUint64(raw[:], math.Float64bits(point.Value))
		payload = append(payload, raw[:]...)
	}
	return validation.Checksum(payload)
}

// ListReadings returns readings for a sensor or chamber inside an optional window.
func (s *Service) ListReadings(filter store.ReadingFilter) ([]model.Reading, error) {
	if filter.Limit <= 0 {
		filter.Limit = 1000
	}
	return s.store.ListReadings(filter)
}

// SiteCompleteness reports accepted telemetry coverage per site over the
// trailing window, including per-sensor detail.
type CompletenessReport struct {
	SiteID       string           `json:"site_id"`
	From         time.Time        `json:"from"`
	To           time.Time        `json:"to"`
	Completeness float64          `json:"completeness"`
	Sensors      []SensorCoverage `json:"sensors"`
	Missing      []string         `json:"missing_sensors"`
}

type SensorCoverage struct {
	SensorID     string  `json:"sensor_id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Accepted     int     `json:"accepted"`
	Expected     int     `json:"expected"`
	Completeness float64 `json:"completeness"`
}

func (s *Service) SiteCompleteness(siteID string, window time.Duration) (CompletenessReport, error) {
	site, err := s.requireSite(siteID)
	if err != nil {
		return CompletenessReport{}, err
	}
	if window <= 0 {
		window = 24 * time.Hour
	}
	to := s.now()
	from := to.Add(-window)
	sensors, err := s.store.ListSensorsBySite(site.ID)
	if err != nil {
		return CompletenessReport{}, err
	}
	counts, err := s.store.CountAcceptedInWindow(site.ID, from, to)
	if err != nil {
		return CompletenessReport{}, err
	}
	report := CompletenessReport{SiteID: site.ID, From: from, To: to}
	totalExpected, totalActual := 0.0, 0.0
	for _, sensor := range sensors {
		intervalHours := time.Duration(sensor.SampleIntervalSec) * time.Second
		expected := int(window / intervalHours)
		actual := counts[sensor.ID]
		ratio := 0.0
		if expected > 0 {
			ratio = float64(actual) / float64(expected)
			if ratio > 1 {
				ratio = 1
			}
		}
		report.Sensors = append(report.Sensors, SensorCoverage{
			SensorID:     sensor.ID,
			Name:         sensor.Name,
			Type:         string(sensor.Type),
			Accepted:     actual,
			Expected:     expected,
			Completeness: ratio,
		})
		totalExpected += float64(expected)
		totalActual += float64(actual)
		if expected > 0 && ratio < 0.5 {
			report.Missing = append(report.Missing, sensor.ID)
		}
	}
	if totalExpected > 0 {
		report.Completeness = totalActual / totalExpected
	}
	if report.Completeness > 1 {
		report.Completeness = 1
	}
	return report, nil
}
