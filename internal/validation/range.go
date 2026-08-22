package validation

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
)

type Limits struct {
	FutureSkew       time.Duration
	RetentionWindow  time.Duration
	MaxBatchReadings int
}

func DefaultLimits() Limits {
	return Limits{
		FutureSkew:       5 * time.Minute,
		RetentionWindow:  30 * 24 * time.Hour,
		MaxBatchReadings: 5000,
	}
}

type RangeError struct {
	SensorID string
	Value    float64
	Min      float64
	Max      float64
}

func (e RangeError) Error() string {
	return fmt.Sprintf("validation: value %.3f outside sensor %s range [%.3f, %.3f]", e.Value, e.SensorID, e.Min, e.Max)
}

var (
	ErrEmptyBatch     = errors.New("validation: telemetry batch has no readings")
	ErrBatchTooLarge  = errors.New("validation: telemetry batch exceeds size limit")
	ErrFutureReading  = errors.New("validation: reading observed in the future")
	ErrStaleReading   = errors.New("validation: reading older than retention window")
	ErrDuplicatePoint = errors.New("validation: duplicate reading in batch")
)

func ValidateBatch(batch model.TelemetryBatch, limits Limits, now time.Time) error {
	if len(batch.Readings) == 0 {
		return ErrEmptyBatch
	}
	if len(batch.Readings) > limits.MaxBatchReadings {
		return ErrBatchTooLarge
	}
	if batch.SiteID == "" || !model.IsID(batch.SiteID) {
		return errors.New("validation: batch site id is missing or malformed")
	}
	if !batch.SentAt.IsZero() && batch.SentAt.After(now.Add(limits.FutureSkew)) {
		return ErrFutureReading
	}
	seen := make(map[string]struct{}, len(batch.Readings))
	for _, point := range batch.Readings {
		key := point.SensorID + "|" + model.EnsureTime(point.ObservedAt).String()
		if _, dup := seen[key]; dup {
			return ErrDuplicatePoint
		}
		seen[key] = struct{}{}
		if err := ValidateTimestamp(point.ObservedAt, limits, now); err != nil {
			return err
		}
		if math.IsNaN(point.Value) || math.IsInf(point.Value, 0) {
			return errors.New("validation: reading value is not finite")
		}
	}
	return nil
}

func ValidateTimestamp(observed time.Time, limits Limits, now time.Time) error {
	if observed.IsZero() {
		return errors.New("validation: observation timestamp is zero")
	}
	if observed.After(now.Add(limits.FutureSkew)) {
		return ErrFutureReading
	}
	if now.Sub(observed) > limits.RetentionWindow {
		return ErrStaleReading
	}
	return nil
}

func CheckRange(sensor model.Sensor, value float64) error {
	if value < sensor.MinValue || value > sensor.MaxValue {
		return RangeError{SensorID: sensor.ID, Value: value, Min: sensor.MinValue, Max: sensor.MaxValue}
	}
	return nil
}

// ClassifyQuality maps a raw telemetry point to a storage quality flag.
// Out-of-range points are kept as rejected for auditing instead of being dropped.
func ClassifyQuality(sensor model.Sensor, raw float64) (model.ReadingQuality, float64) {
	value := sensor.Correct(raw)
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return model.QualityRejected, 0
	}
	switch {
	case value < sensor.MinValue || value > sensor.MaxValue:
		return model.QualityRejected, value
	case nearBoundary(value, sensor.MinValue) || nearBoundary(value, sensor.MaxValue):
		return model.QualitySuspect, value
	default:
		return model.QualityGood, value
	}
}

func nearBoundary(value, boundary float64) bool {
	span := math.Abs(boundary) * 0.0000001
	return math.Abs(value-boundary) <= span
}

func RequireText(field, value string, maxLen int) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("validation: %s is required", field)
	}
	if len(trimmed) > maxLen {
		return fmt.Errorf("validation: %s exceeds %d characters", field, maxLen)
	}
	return nil
}
