package store

import (
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
)

func (s *Store) CreateAlert(alert model.Alert) error {
	_, err := s.db.Exec(
		`INSERT INTO alerts(id, site_id, chamber_id, sensor_id, kind, severity, status,
		   dedup_key, message, first_seen_at, last_seen_at, acknowledged_at, closed_at, occurrences)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		alert.ID, alert.SiteID, alert.ChamberID, alert.SensorID, string(alert.Kind),
		string(alert.Severity), string(alert.Status), alert.DedupKey, alert.Message,
		formatTime(alert.FirstSeenAt), formatTime(alert.LastSeenAt),
		formatTime(alert.AcknowledgedAt), formatTime(alert.ClosedAt), alert.Occurrences,
	)
	if isUniqueViolation(err) {
		return ErrDuplicateAlert
	}
	return wrap("create alert", err)
}

func (s *Store) GetActiveAlertByDedupKey(dedupKey string) (model.Alert, error) {
	row := s.db.QueryRow(alertColumns+` FROM alerts WHERE dedup_key = ? AND status != 'closed'`, dedupKey)
	return scanAlert(row)
}

func (s *Store) BumpAlert(id string, severity model.AlertSeverity, message string, now time.Time) error {
	_, err := s.db.Exec(
		`UPDATE alerts SET occurrences = occurrences + 1, severity = ?, message = ?, last_seen_at = ?
		 WHERE id = ?`,
		string(severity), message, formatTime(now), id)
	return wrap("bump alert", err)
}

func (s *Store) AcknowledgeAlert(id string, now time.Time) error {
	res, err := s.db.Exec(
		`UPDATE alerts SET status = 'acknowledged', acknowledged_at = ?
		 WHERE id = ? AND status = 'open'`, formatTime(now), id)
	if err != nil {
		return wrap("acknowledge alert", err)
	}
	return requireAffected(res, "alert")
}

func (s *Store) CloseAlert(id string, now time.Time) error {
	res, err := s.db.Exec(
		`UPDATE alerts SET status = 'closed', closed_at = ?
		 WHERE id = ? AND status != 'closed'
		   AND (severity != 'critical' OR status = 'acknowledged')`,
		formatTime(now), id)
	if err != nil {
		return wrap("close alert", err)
	}
	return requireAffected(res, "alert")
}

func (s *Store) ListAlerts(siteID, chamberID string, status string, limit int) ([]model.Alert, error) {
	query := alertColumns + ` FROM alerts WHERE 1=1`
	var args []any
	if siteID != "" {
		query += ` AND site_id = ?`
		args = append(args, siteID)
	}
	if chamberID != "" {
		query += ` AND chamber_id = ?`
		args = append(args, chamberID)
	}
	switch status {
	case "active":
		query += ` AND status != 'closed'`
	case "open":
		query += ` AND status = 'open'`
	case "acked", "acknowledged":
		query += ` AND status = 'acknowledged'`
	case "closed":
		query += ` AND status = 'closed'`
	case "":
	default:
		query += ` AND status = ?`
		args = append(args, status)
	}
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	query += ` ORDER BY first_seen_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, wrap("list alerts", err)
	}
	defer rows.Close()
	var alerts []model.Alert
	for rows.Next() {
		alert, scanErr := scanAlert(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		alerts = append(alerts, alert)
	}
	return alerts, rows.Err()
}

func (s *Store) CountActiveAlerts(siteID string) (total int, critical int, err error) {
	err = s.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN severity='critical' THEN 1 ELSE 0 END),0)
		 FROM alerts WHERE site_id = ? AND status != 'closed'`, siteID).Scan(&total, &critical)
	return total, critical, wrap("count active alerts", err)
}

// ExpireStaleAlerts closes open/acked alerts whose dedup key has not been seen
// within the given quiet window. Returns the number of closed alerts.
func (s *Store) ExpireStaleAlerts(quiet time.Duration, now time.Time) (int64, error) {
	res, err := s.db.Exec(
		`UPDATE alerts SET status = 'closed', closed_at = ?
		 WHERE status != 'closed' AND last_seen_at < ?`,
		formatTime(now), formatTime(now.Add(-quiet)))
	if err != nil {
		return 0, wrap("expire stale alerts", err)
	}
	closed, err := res.RowsAffected()
	return closed, wrap("expire stale alerts rows", err)
}

const alertColumns = `SELECT id, site_id, chamber_id, sensor_id, kind, severity, status,
	dedup_key, message, first_seen_at, last_seen_at, acknowledged_at, closed_at, occurrences`

func scanAlert(row rowScanner) (model.Alert, error) {
	var (
		alert          model.Alert
		kind           string
		severity       string
		status         string
		firstSeen      string
		lastSeen       string
		acknowledgedAt string
		closedAt       string
	)
	if err := row.Scan(&alert.ID, &alert.SiteID, &alert.ChamberID, &alert.SensorID, &kind,
		&severity, &status, &alert.DedupKey, &alert.Message, &firstSeen, &lastSeen,
		&acknowledgedAt, &closedAt, &alert.Occurrences); err != nil {
		return model.Alert{}, mapNotFound(err)
	}
	alert.Kind = model.AlertKind(kind)
	alert.Severity = model.AlertSeverity(severity)
	alert.Status = model.AlertStatus(status)
	alert.FirstSeenAt = parseTime(firstSeen)
	alert.LastSeenAt = parseTime(lastSeen)
	alert.AcknowledgedAt = parseTime(acknowledgedAt)
	alert.ClosedAt = parseTime(closedAt)
	return alert, nil
}

func (s *Store) GetAlert(id string) (model.Alert, error) {
	row := s.db.QueryRow(alertColumns+` FROM alerts WHERE id = ?`, id)
	return scanAlert(row)
}
