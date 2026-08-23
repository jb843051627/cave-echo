package engine

import (
	"fmt"
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
)

type ChamberInput struct {
	Chamber      model.Chamber
	Readings     []model.Reading // latest reading per sensor type
	GasSample    *model.GasSample
	DripTrend    model.DripTrend
	DripEvents   []model.DripEvent
	ExpectedRate float64 // expected accepted readings per hour across sensors
	ActualRate   float64 // observed accepted readings per hour
	Now          time.Time
}

// Assess synthesises the per-chamber stability record. Every risk dimension
// is scored in 0..1 and combined into an overall level:
//
//	stable     all risks low, telemetry healthy
//	watch      at least one elevated risk or incomplete telemetry
//	restricted multiple risks or a critical gas signal
//	closed     condensation plus gas hazard, or completeness collapse
func (e *Engine) Assess(input ChamberInput) model.StabilityAssessment {
	tempC := pickReading(input.Readings, model.SensorTemperature)
	humidity := pickReading(input.Readings, model.SensorHumidity)
	airflow := findReading(input.Readings, model.SensorAirflow)

	condensation := 0.0
	if tempC != nil && humidity != nil {
		condensation = e.CondensationRisk(tempC.Value, humidity.Value)
	} else {
		condensation = 0.4
	}

	gasRisk := 0.0
	if input.GasSample != nil {
		gasRisk = e.GasRisk(*input.GasSample)
	}

	dripRisk := e.DripRisk(input.DripTrend, input.DripEvents)
	airflowRisk := e.AirflowRisk(airflow, input.Chamber.AirflowDirection)

	completeness := 1.0
	if input.ExpectedRate > 0 {
		completeness = clamp01(input.ActualRate / input.ExpectedRate)
	}

	score := 100.0
	score -= condensation * 30
	score -= gasRisk * 30
	score -= dripRisk * 15
	score -= airflowRisk * 10
	score -= (1 - completeness) * 15
	if score < 0 {
		score = 0
	}

	level := e.levelFor(condensation, gasRisk, dripRisk, airflowRisk, completeness)
	return model.StabilityAssessment{
		SiteID:           input.Chamber.SiteID,
		ChamberID:        input.Chamber.ID,
		AssessedAt:       model.EnsureTime(input.Now),
		Score:            round2(score),
		CondensationRisk: round2(condensation),
		GasRisk:          round2(gasRisk),
		DripRisk:         round2(dripRisk),
		AirflowRisk:      round2(airflowRisk),
		Completeness:     round2(completeness),
		Level:            level,
		Summary:          e.summarise(input.Chamber.Name, level, condensation, gasRisk, dripRisk, completeness),
	}
}

func (e *Engine) levelFor(condensation, gasRisk, dripRisk, airflowRisk, completeness float64) model.StabilityLevel {
	criticalGas := gasRisk >= 0.8
	elevated := 0
	for _, risk := range []float64{condensation, gasRisk, dripRisk, airflowRisk} {
		if risk >= 0.6 {
			elevated++
		}
	}
	switch {
	case criticalGas && condensation >= 0.6:
		return model.StabilityClosed
	case criticalGas || elevated >= 2:
		return model.StabilityRestricted
	case condensation >= 0.6 || gasRisk >= 0.5 || dripRisk >= 0.6 || airflowRisk >= 0.7 || completeness < 0.5:
		return model.StabilityWatch
	default:
		return model.StabilityStable
	}
}

func (e *Engine) summarise(chamber string, level model.StabilityLevel, condensation, gasRisk, dripRisk, completeness float64) string {
	base := fmt.Sprintf("%s assessed %s", chamber, level)
	switch {
	case condensation >= 0.6:
		base += "; condensation imminent on formations"
	}
	switch {
	case gasRisk >= 0.8:
		base += "; gas hazard severe"
	case gasRisk >= 0.5:
		base += "; CO2 trending toward limits"
	}
	if dripRisk >= 0.6 {
		base += "; drip regime anomalous"
	}
	if completeness < 0.5 {
		base += "; telemetry coverage degraded"
	}
	return base
}

// ThresholdBreach inspects one corrected reading against its sensor's warning
// and critical thresholds and proposes an alert severity.
func (e *Engine) ThresholdBreach(sensor model.Sensor, value float64) model.AlertSeverity {
	if sensor.WarningThreshold == 0 && sensor.CriticalThreshold == 0 {
		return ""
	}
	if sensor.CriticalThreshold != 0 && value >= sensor.CriticalThreshold {
		return model.SeverityCritical
	}
	if sensor.WarningThreshold != 0 && value >= sensor.WarningThreshold {
		return model.SeverityWarning
	}
	return ""
}

func pickReading(readings []model.Reading, kind model.SensorType) *model.Reading {
	return findReading(readings, kind)
}

func findReading(readings []model.Reading, kind model.SensorType) *model.Reading {
	for i := range readings {
		if readings[i].SensorType == kind {
			return &readings[i]
		}
	}
	return nil
}
