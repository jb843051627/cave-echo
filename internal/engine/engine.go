package engine

import (
	"math"

	"github.com/jb843051627/cave-echo/internal/model"
)

// Engine holds the pure microclimate computation logic. It is stateless apart
// from tuning knobs, so it can be shared by the service layer freely.
type Engine struct {
	dewPointBeta   float64
	co2WarningPPM  float64
	co2CriticalPPM float64
	oxygenFloor    float64
	radonCeiling   float64
	dripSlopeAlert float64
	batSeasonStart int
	batSeasonEnd   int
}

func New() *Engine {
	return &Engine{
		dewPointBeta:   17.62,
		co2WarningPPM:  2500,
		co2CriticalPPM: 5000,
		oxygenFloor:    19.5,
		radonCeiling:   300,
		dripSlopeAlert: 5.0,
		batSeasonStart: 8,
		batSeasonEnd:   11,
	}
}

// DewPoint uses the Magnus approximation with temperature in Celsius and
// relative humidity in percent.
func (e *Engine) DewPoint(tempC, humidity float64) float64 {
	if humidity <= 0 {
		return -40
	}
	if humidity > 100 {
		humidity = 100
	}
	gamma := math.Log(humidity/100) + (e.dewPointBeta*tempC)/(243.12+tempC)
	return (243.12 * gamma) / (e.dewPointBeta - gamma)
}

// CondensationRisk scores how likely water is condensing on formations.
// risk = 0 when the dew point gap is large, approaching 1 as surfaces cool
// toward the dew point.
func (e *Engine) CondensationRisk(tempC, humidity float64) float64 {
	gap := tempC - e.DewPoint(tempC, humidity)
	switch {
	case gap <= 0:
		return 1
	case gap >= 4:
		return 0
	default:
		return clamp01(1 - gap/4)
	}
}

// GasRisk combines CO2 concentration, oxygen depletion and radon into a
// single 0..1 score. The heaviest term dominates but weaker signals still
// lift the score so mixed hazards surface earlier.
func (e *Engine) GasRisk(sample model.GasSample) float64 {
	co2Score := clamp01((sample.CO2PPM - e.co2WarningPPM/2) / (e.co2CriticalPPM - e.co2WarningPPM/2))
	oxygenScore := clamp01((e.oxygenFloor + 1.5 - sample.OxygenPercent) / 3)
	radonScore := clamp01(sample.RadonBqM3 / (e.radonCeiling * 2))
	score := co2Score*0.55 + oxygenScore*0.30 + radonScore*0.15
	return clamp01(score)
}

// DripTrend fits a least squares line over drip rates and reports the slope
// in drips per minute per day together with a direction label.
func (e *Engine) DripTrend(events []model.DripEvent) model.DripTrend {
	if len(events) == 0 {
		return model.DripTrend{Direction: "unknown"}
	}
	ordered := make([]model.DripEvent, len(events))
	copy(ordered, events)
	sortByObserved(ordered)

	var sumX, sumY, sumXY, sumXX float64
	n := float64(len(ordered))
	for _, event := range ordered {
		x := event.ObservedAt.Sub(ordered[0].ObservedAt).Hours() / 24
		y := event.RatePerMinute
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}
	denominator := n*sumXX - sumX*sumX
	slope := (n*sumXY - sumX*sumY) / denominator

	total := 0.0
	recentCount := 0
	cutoff := len(ordered) - len(ordered)/3
	if cutoff < 0 {
		cutoff = 0
	}
	for i, event := range ordered {
		total += event.RatePerMinute
		if i >= cutoff {
			recentCount++
		}
	}
	recentTotal := 0.0
	for i := cutoff; i < len(ordered); i++ {
		recentTotal += ordered[i].RatePerMinute
	}
	averageRate := total / n
	recentRate := averageRate
	if recentCount > 0 {
		recentRate = recentTotal / float64(recentCount)
	}
	if len(ordered) == 1 {
		recentRate = ordered[0].RatePerMinute
	}

	direction := "stable"
	switch {
	case slope > e.dripSlopeAlert:
		direction = "rising"
	case slope < -e.dripSlopeAlert:
		direction = "falling"
	}
	return model.DripTrend{
		AverageRate:     round2(averageRate),
		RecentRate:      round2(recentRate),
		SlopePerDay:     round2(slope),
		EventCount:      len(ordered),
		ObservationFrom: ordered[0].ObservedAt,
		ObservationTo:   ordered[len(ordered)-1].ObservedAt,
		Direction:       direction,
	}
}

// DripRisk scores abnormal drip behaviour. A rising slope or cloudy/amber
// coloration both push the score up; a single event window yields a neutral
// baseline so sparse chambers are not punished.
func (e *Engine) DripRisk(trend model.DripTrend, events []model.DripEvent) float64 {
	if trend.EventCount == 0 {
		return 0
	}
	risk := clamp01(math.Abs(trend.SlopePerDay) / (e.dripSlopeAlert * 4))
	colored := 0
	for _, event := range events {
		if event.Color != model.DripClear {
			colored++
		}
	}
	colorShare := 0.0
	if len(events) > 0 {
		colorShare = float64(colored) / float64(len(events))
	}
	risk = clamp01(risk*0.6 + colorShare*0.4)
	if trend.Direction == "rising" {
		risk = clamp01(risk + 0.15)
	}
	return risk
}

// AirflowRisk compares the latest airflow reading against the chamber's
// declared airflow direction band; missing data counts as mild risk because
// ventilation state is unknown.
func (e *Engine) AirflowRisk(latest *model.Reading, expectedBand string) float64 {
	if latest == nil {
		return 0.3
	}
	deviation := math.Abs(latest.Value)
	switch expectedBand {
	case "still":
		return clamp01(deviation / 2)
	case "moderate":
		return clamp01(math.Abs(deviation-1) / 3)
	case "draft":
		return clamp01((3 - deviation) / 3)
	default:
		return clamp01(deviation / 4)
	}
}

func clamp01(value float64) float64 {
	if math.IsNaN(value) {
		return 0
	}
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func sortByObserved(events []model.DripEvent) {
	for i := 1; i < len(events); i++ {
		for j := i; j > 0 && events[j].ObservedAt.Before(events[j-1].ObservedAt); j-- {
			events[j], events[j-1] = events[j-1], events[j]
		}
	}
}
