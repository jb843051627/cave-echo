package model

import "time"

type StabilityAssessment struct {
	ID               string         `json:"id"`
	SiteID           string         `json:"site_id"`
	ChamberID        string         `json:"chamber_id"`
	AssessedAt       time.Time      `json:"assessed_at"`
	Score            float64        `json:"score"`
	CondensationRisk float64        `json:"condensation_risk"`
	GasRisk          float64        `json:"gas_risk"`
	DripRisk         float64        `json:"drip_risk"`
	AirflowRisk      float64        `json:"airflow_risk"`
	Completeness     float64        `json:"completeness"`
	Level            StabilityLevel `json:"level"`
	Summary          string         `json:"summary"`
	CreatedAt        time.Time      `json:"created_at"`
}

func (a StabilityAssessment) RequiresRestriction() bool {
	return a.Level == StabilityRestricted || a.Level == StabilityClosed
}

func (a StabilityAssessment) RiskNames() []string {
	result := make([]string, 0, 4)
	if a.CondensationRisk >= 0.6 {
		result = append(result, "condensation")
	}
	if a.GasRisk >= 0.6 {
		result = append(result, "gas")
	}
	if a.DripRisk >= 0.6 {
		result = append(result, "drip")
	}
	if a.AirflowRisk >= 0.6 {
		result = append(result, "airflow")
	}
	return result
}
