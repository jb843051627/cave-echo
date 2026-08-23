package store

import (
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
)

func (s *Store) CreateDripEvent(event model.DripEvent) error {
	_, err := s.db.Exec(
		`INSERT INTO drip_events(id, chamber_id, observed_at, rate_per_minute, mineralization,
		   color, location, duration_seconds, observer, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?)`,
		event.ID, event.ChamberID, formatTime(event.ObservedAt), event.RatePerMinute,
		event.Mineralization, string(event.Color), event.Location,
		event.DurationSeconds, event.Observer, formatTime(event.CreatedAt),
	)
	return wrap("create drip event", err)
}

func (s *Store) ListDripEvents(chamberID string, from, to time.Time, limit int) ([]model.DripEvent, error) {
	query := `SELECT id, chamber_id, observed_at, rate_per_minute, mineralization, color,
		location, duration_seconds, observer, created_at FROM drip_events WHERE 1=1`
	var args []any
	if chamberID != "" {
		query += ` AND chamber_id = ?`
		args = append(args, chamberID)
	}
	if !from.IsZero() {
		query += ` AND observed_at >= ?`
		args = append(args, formatTime(from))
	}
	if !to.IsZero() {
		query += ` AND observed_at <= ?`
		args = append(args, formatTime(to))
	}
	if limit <= 0 || limit > 5000 {
		limit = 5000
	}
	query += ` ORDER BY observed_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, wrap("list drip events", err)
	}
	defer rows.Close()
	var events []model.DripEvent
	for rows.Next() {
		event, scanErr := scanDrip(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) CountDripsForChamber(chamberID string, from, to time.Time) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM drip_events WHERE chamber_id = ? AND observed_at BETWEEN ? AND ?`,
		chamberID, formatTime(from), formatTime(to)).Scan(&count)
	return count, wrap("count drips", err)
}

func scanDrip(row rowScanner) (model.DripEvent, error) {
	var (
		event     model.DripEvent
		color     string
		observed  string
		createdAt string
	)
	if err := row.Scan(&event.ID, &event.ChamberID, &observed, &event.RatePerMinute,
		&event.Mineralization, &color, &event.Location, &event.DurationSeconds,
		&event.Observer, &createdAt); err != nil {
		return model.DripEvent{}, mapNotFound(err)
	}
	event.Color = model.DripColor(color)
	event.ObservedAt = parseTime(observed)
	event.CreatedAt = parseTime(createdAt)
	return event, nil
}
