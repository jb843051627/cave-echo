package engine

import (
	"fmt"
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
)

// BatSeasonActive reports whether the given month falls inside the
// hibernation / maternity window when visitor access should be reduced.
func (e *Engine) BatSeasonActive(at time.Time) bool {
	month := int(at.UTC().Month())
	return month > e.batSeasonStart && month < e.batSeasonEnd
}

// ProtectionMessage renders the seasonal guidance for a chamber.
func (e *Engine) ProtectionMessage(chamber model.Chamber, at time.Time) string {
	if !chamber.BatHabitat {
		return ""
	}
	if e.BatSeasonActive(at) {
		return fmt.Sprintf("%s is a bat habitat; hibernation season active, keep visits out", chamber.Name)
	}
	return fmt.Sprintf("%s is a bat habitat; post-season monitoring recommended", chamber.Name)
}

// SeasonalNotices collects site-level protection messages.
func (e *Engine) SeasonalNotices(chambers []model.Chamber, at time.Time) []string {
	var notices []string
	for _, chamber := range chambers {
		if message := e.ProtectionMessage(chamber, at); message != "" && e.BatSeasonActive(at) {
			notices = append(notices, message)
		}
	}
	return notices
}

// SurveyStageGuard blocks stage transitions that would violate the bat
// season: fieldwork may not start inside the season for bat habitats unless
// the chamber carries an explicit research exception rule.
func (e *Engine) SurveyStageGuard(chamber model.Chamber, from, to model.SurveyStage, at time.Time) error {
	if to != model.SurveyFieldwork || from != model.SurveyPlanned {
		return nil
	}
	if !chamber.BatHabitat {
		return nil
	}
	if chamber.ProtectionRule == "research_exception" {
		return nil
	}
	if e.BatSeasonActive(at) {
		return fmt.Errorf("engine: fieldwork on %s blocked during bat season", chamber.Name)
	}
	return nil
}

type ProposedAlert struct {
	SiteID    string
	ChamberID string
	SensorID  string
	Kind      model.AlertKind
	Severity  model.AlertSeverity
	Message   string
	Rule      string
}

// AssessmentAlerts derives alerts from a fresh assessment. Each rule has a
// stable name that feeds into the dedup key so repeated evaluations within
// the dedup window collapse onto one alert.
func (e *Engine) AssessmentAlerts(assessment model.StabilityAssessment) []ProposedAlert {
	var alerts []ProposedAlert
	if assessment.CondensationRisk >= 0.7 {
		alerts = append(alerts, ProposedAlert{
			SiteID:    assessment.SiteID,
			ChamberID: assessment.ChamberID,
			Kind:      model.AlertMicroclimate,
			Severity:  model.SeverityWarning,
			Message:   fmt.Sprintf("condensation risk %.2f in chamber %s", assessment.CondensationRisk, assessment.ChamberID),
			Rule:      "condensation_high",
		})
	}
	switch {
	case assessment.GasRisk >= 0.8:
		alerts = append(alerts, ProposedAlert{
			SiteID:    assessment.SiteID,
			ChamberID: assessment.ChamberID,
			Kind:      model.AlertGas,
			Severity:  model.SeverityCritical,
			Message:   fmt.Sprintf("gas risk %.2f exceeds critical bound in chamber %s", assessment.GasRisk, assessment.ChamberID),
			Rule:      "gas_critical",
		})
	case assessment.GasRisk >= 0.5:
		alerts = append(alerts, ProposedAlert{
			SiteID:    assessment.SiteID,
			ChamberID: assessment.ChamberID,
			Kind:      model.AlertGas,
			Severity:  model.SeverityWarning,
			Message:   fmt.Sprintf("gas risk %.2f elevated in chamber %s", assessment.GasRisk, assessment.ChamberID),
			Rule:      "gas_elevated",
		})
	}
	if assessment.DripRisk >= 0.7 {
		alerts = append(alerts, ProposedAlert{
			SiteID:    assessment.SiteID,
			ChamberID: assessment.ChamberID,
			Kind:      model.AlertDrip,
			Severity:  model.SeverityWarning,
			Message:   fmt.Sprintf("drip regime anomaly score %.2f in chamber %s", assessment.DripRisk, assessment.ChamberID),
			Rule:      "drip_anomaly",
		})
	}
	if assessment.Completeness < 0.4 {
		alerts = append(alerts, ProposedAlert{
			SiteID:    assessment.SiteID,
			ChamberID: assessment.ChamberID,
			Kind:      model.AlertCompleteness,
			Severity:  model.SeverityInfo,
			Message:   fmt.Sprintf("telemetry completeness %.2f below floor in chamber %s", assessment.Completeness, assessment.ChamberID),
			Rule:      "completeness_low",
		})
	}
	return alerts
}

// SensorOfflineAlert builds the offline proposal for a silent sensor.
func SensorOfflineAlert(sensor model.Sensor, now time.Time) ProposedAlert {
	return ProposedAlert{
		SiteID:    sensor.SiteID,
		ChamberID: sensor.ChamberID,
		SensorID:  sensor.ID,
		Kind:      model.AlertSensor,
		Severity:  model.SeverityWarning,
		Message:   fmt.Sprintf("sensor %s (%s) silent since %s", sensor.Name, sensor.Type, formatStamp(sensor.LastHeartbeatAt)),
		Rule:      "sensor_offline",
	}
}

func formatStamp(value time.Time) string {
	if value.IsZero() {
		return "never"
	}
	return value.UTC().Format(time.RFC3339)
}
