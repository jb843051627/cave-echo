package store

import (
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
)

func (s *Store) CreateSurvey(survey model.Survey) error {
	_, err := s.db.Exec(
		`INSERT INTO surveys(id, site_id, chamber_id, transect, surface_condition,
		   crystal_change_mm, stage, started_at, completed_at, findings, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		survey.ID, survey.SiteID, survey.ChamberID, survey.Transect, survey.SurfaceCondition,
		survey.CrystalChangeMM, string(survey.Stage), formatTime(survey.StartedAt),
		formatTime(survey.CompletedAt), survey.Findings,
		formatTime(survey.CreatedAt), formatTime(survey.UpdatedAt),
	)
	return wrap("create survey", err)
}

func (s *Store) GetSurvey(id string) (model.Survey, error) {
	row := s.db.QueryRow(surveyColumns+` FROM surveys WHERE id = ?`, id)
	return scanSurvey(row)
}

func (s *Store) ListSurveys(siteID string, activeOnly bool) ([]model.Survey, error) {
	query := surveyColumns + ` FROM surveys WHERE 1=1`
	var args []any
	if siteID != "" {
		query += ` AND site_id = ?`
		args = append(args, siteID)
	}
	if activeOnly {
		query += ` AND stage != ?`
		args = append(args, string(model.SurveyClosed))
	}
	query += ` ORDER BY started_at DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, wrap("list surveys", err)
	}
	defer rows.Close()
	var surveys []model.Survey
	for rows.Next() {
		survey, scanErr := scanSurvey(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		surveys = append(surveys, survey)
	}
	return surveys, rows.Err()
}

// TransitionSurvey moves a survey to the next stage inside a transaction.
// The caller is responsible for validating the state machine; the store
// re-checks optimistic concurrency via the previous stage.
func (s *Store) TransitionSurvey(id string, from, to model.SurveyStage, findings string, now time.Time) error {
	return wrap("transition survey", s.InTx(func(tx *sqlTx) error {
		res, err := tx.Exec(
			`UPDATE surveys SET stage = ?, findings = CASE WHEN ? = '' THEN findings ELSE ? END,
			   updated_at = ?, completed_at = CASE WHEN ? = 'closed' THEN ? ELSE completed_at END
			 WHERE id = ? AND stage = ?`,
			string(to), findings, findings, formatTime(now), string(to), formatTime(now), id, string(from))
		if err != nil {
			return err
		}
		return requireAffected(res, "survey")
	}))
}

const surveyColumns = `SELECT id, site_id, chamber_id, transect, surface_condition,
	crystal_change_mm, stage, started_at, completed_at, findings, created_at, updated_at`

func scanSurvey(row rowScanner) (model.Survey, error) {
	var (
		survey      model.Survey
		stage       string
		startedAt   string
		completedAt string
		createdAt   string
		updatedAt   string
	)
	if err := row.Scan(&survey.ID, &survey.SiteID, &survey.ChamberID, &survey.Transect,
		&survey.SurfaceCondition, &survey.CrystalChangeMM, &stage, &startedAt, &completedAt,
		&survey.Findings, &createdAt, &updatedAt); err != nil {
		return model.Survey{}, mapNotFound(err)
	}
	survey.Stage = model.SurveyStage(stage)
	survey.StartedAt = parseTime(startedAt)
	survey.CompletedAt = parseTime(completedAt)
	survey.CreatedAt = parseTime(createdAt)
	survey.UpdatedAt = parseTime(updatedAt)
	return survey, nil
}
