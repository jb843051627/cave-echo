package store

import (
	"github.com/jb843051627/cave-echo/internal/model"
)

func (s *Store) CreateAssessment(assessment model.StabilityAssessment) error {
	_, err := s.db.Exec(
		`INSERT INTO stability_assessments(id, site_id, chamber_id, assessed_at, score,
		   condensation_risk, gas_risk, drip_risk, airflow_risk, completeness, level, summary, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		assessment.ID, assessment.SiteID, assessment.ChamberID, formatTime(assessment.AssessedAt),
		assessment.Score, assessment.CondensationRisk, assessment.GasRisk, assessment.DripRisk,
		assessment.AirflowRisk, assessment.Completeness, string(assessment.Level),
		assessment.Summary, formatTime(assessment.CreatedAt),
	)
	return wrap("create assessment", err)
}

func (s *Store) LatestAssessment(chamberID string) (model.StabilityAssessment, error) {
	row := s.db.QueryRow(assessmentColumns+` FROM stability_assessments
		WHERE chamber_id = ? ORDER BY assessed_at DESC LIMIT 1`, chamberID)
	return scanAssessment(row)
}

func (s *Store) ListAssessments(chamberID string, siteID string, limit int) ([]model.StabilityAssessment, error) {
	query := assessmentColumns + ` FROM stability_assessments WHERE 1=1`
	var args []any
	if chamberID != "" {
		query += ` AND chamber_id = ?`
		args = append(args, chamberID)
	}
	if siteID != "" {
		query += ` AND site_id = ?`
		args = append(args, siteID)
	}
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	query += ` ORDER BY assessed_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, wrap("list assessments", err)
	}
	defer rows.Close()
	var assessments []model.StabilityAssessment
	for rows.Next() {
		assessment, scanErr := scanAssessment(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		assessments = append(assessments, assessment)
	}
	return assessments, rows.Err()
}

func (s *Store) CountRestrictedChambers(siteID string) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(DISTINCT a.chamber_id) FROM stability_assessments a
		 JOIN (SELECT chamber_id, MAX(assessed_at) AS latest FROM stability_assessments
		       WHERE site_id = ? GROUP BY chamber_id) latest
		   ON a.chamber_id = latest.chamber_id AND a.assessed_at = latest.latest
		 WHERE a.level IN ('restricted','closed')`, siteID).Scan(&count)
	return count, wrap("count restricted chambers", err)
}

const assessmentColumns = `SELECT id, site_id, chamber_id, assessed_at, score,
	condensation_risk, gas_risk, drip_risk, airflow_risk, completeness, level, summary, created_at`

func scanAssessment(row rowScanner) (model.StabilityAssessment, error) {
	var (
		assessment model.StabilityAssessment
		level      string
		assessedAt string
		createdAt  string
	)
	if err := row.Scan(&assessment.ID, &assessment.SiteID, &assessment.ChamberID, &assessedAt,
		&assessment.Score, &assessment.CondensationRisk, &assessment.GasRisk, &assessment.DripRisk,
		&assessment.AirflowRisk, &assessment.Completeness, &level, &assessment.Summary,
		&createdAt); err != nil {
		return model.StabilityAssessment{}, mapNotFound(err)
	}
	assessment.Level = model.StabilityLevel(level)
	assessment.AssessedAt = parseTime(assessedAt)
	assessment.CreatedAt = parseTime(createdAt)
	return assessment, nil
}
