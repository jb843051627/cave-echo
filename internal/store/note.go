package store

import (
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
)

func (s *Store) CreateNote(note model.ConservationNote) error {
	_, err := s.db.Exec(
		`INSERT INTO conservation_notes(id, site_id, chamber_id, survey_id, author, category,
		   note, action_outcome, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		note.ID, note.SiteID, note.ChamberID, note.SurveyID, note.Author,
		string(note.Category), note.Note, note.ActionOutcome, formatTime(note.CreatedAt),
	)
	return wrap("create note", err)
}

func (s *Store) ListNotes(siteID, chamberID string, limit int) ([]model.ConservationNote, error) {
	query := `SELECT id, site_id, chamber_id, survey_id, author, category, note,
		action_outcome, created_at FROM conservation_notes WHERE 1=1`
	var args []any
	if siteID != "" {
		query += ` AND site_id = ?`
		args = append(args, siteID)
	}
	if chamberID != "" {
		query += ` AND chamber_id = ?`
		args = append(args, chamberID)
	}
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, wrap("list notes", err)
	}
	defer rows.Close()
	var notes []model.ConservationNote
	for rows.Next() {
		note, scanErr := scanNote(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

func (s *Store) ListNotesBySurvey(surveyID string) ([]model.ConservationNote, error) {
	rows, err := s.db.Query(
		`SELECT id, site_id, chamber_id, survey_id, author, category, note,
		   action_outcome, created_at FROM conservation_notes
		 WHERE survey_id = ? ORDER BY created_at`, surveyID)
	if err != nil {
		return nil, wrap("list survey notes", err)
	}
	defer rows.Close()
	var notes []model.ConservationNote
	for rows.Next() {
		note, scanErr := scanNote(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

func (s *Store) CountNotesSince(chamberID string, since time.Time) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM conservation_notes WHERE chamber_id = ? AND created_at >= ?`,
		chamberID, formatTime(since)).Scan(&count)
	return count, wrap("count notes", err)
}

func scanNote(row rowScanner) (model.ConservationNote, error) {
	var (
		note      model.ConservationNote
		category  string
		createdAt string
	)
	if err := row.Scan(&note.ID, &note.SiteID, &note.ChamberID, &note.SurveyID, &note.Author,
		&category, &note.Note, &note.ActionOutcome, &createdAt); err != nil {
		return model.ConservationNote{}, mapNotFound(err)
	}
	note.Category = model.NoteCategory(category)
	note.CreatedAt = parseTime(createdAt)
	return note, nil
}
