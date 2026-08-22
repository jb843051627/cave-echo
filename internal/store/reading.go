package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
)

type ReadingFilter struct {
	SensorID  string
	SiteID    string
	ChamberID string
	Type      model.SensorType
	Quality   model.ReadingQuality
	From      time.Time
	To        time.Time
	Limit     int
}

func (s *Store) InsertReadings(readings []model.Reading) (int, error) {
	if len(readings) == 0 {
		return 0, nil
	}
	inserted := 0
	err := s.InTx(func(tx *sqlTx) error {
		stmt, err := tx.Prepare(
			`INSERT OR IGNORE INTO readings(id, sensor_id, site_id, chamber_id, sensor_type,
			   observed_at, raw_value, value, unit, quality, batch_id, checksum, received_at)
			 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, reading := range readings {
			res, execErr := stmt.Exec(
				reading.ID, reading.SensorID, reading.SiteID, reading.ChamberID, string(reading.SensorType),
				formatTime(reading.ObservedAt), reading.RawValue, reading.Value, reading.Unit,
				string(reading.Quality), reading.BatchID, int64(reading.Checksum), formatTime(reading.ReceivedAt),
			)
			if execErr != nil {
				return execErr
			}
			if _, affErr := res.RowsAffected(); affErr != nil {
				return affErr
			}
			inserted++
		}
		return nil
	})
	if err != nil {
		return inserted, wrap("insert readings", err)
	}
	return inserted, nil
}

func (s *Store) ListReadings(filter ReadingFilter) ([]model.Reading, error) {
	query := strings.Builder{}
	query.WriteString(`SELECT id, sensor_id, site_id, chamber_id, sensor_type, observed_at,
		raw_value, value, unit, quality, batch_id, checksum, received_at FROM readings WHERE 1=1`)
	var args []any
	if filter.SensorID != "" {
		query.WriteString(` AND sensor_id = ?`)
		args = append(args, filter.SensorID)
	}
	if filter.SiteID != "" {
		query.WriteString(` AND site_id = ?`)
		args = append(args, filter.SiteID)
	}
	if filter.ChamberID != "" {
		query.WriteString(` AND chamber_id = ?`)
		args = append(args, filter.ChamberID)
	}
	if filter.Type.Valid() {
		query.WriteString(` AND sensor_type = ?`)
		args = append(args, string(filter.Type))
	}
	if filter.Quality.Valid() {
		query.WriteString(` AND quality = ?`)
		args = append(args, string(filter.Quality))
	}
	if !filter.From.IsZero() {
		query.WriteString(` AND observed_at >= ?`)
		args = append(args, formatTime(filter.From))
	}
	if !filter.To.IsZero() {
		query.WriteString(` AND observed_at <= ?`)
		args = append(args, formatTime(filter.To))
	}
	query.WriteString(` ORDER BY observed_at DESC`)
	limit := filter.Limit
	if limit <= 0 || limit > maxReadingLimit {
		limit = maxReadingLimit
	}
	query.WriteString(fmt.Sprintf(` LIMIT %d`, limit))

	rows, err := s.db.Query(query.String(), args...)
	if err != nil {
		return nil, wrap("list readings", err)
	}
	defer rows.Close()
	var readings []model.Reading
	for rows.Next() {
		reading, scanErr := scanReading(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		readings = append(readings, reading)
	}
	return readings, rows.Err()
}

// CountAcceptedInWindow counts accepted (non-rejected) readings per sensor in a window.
func (s *Store) CountAcceptedInWindow(siteID string, from, to time.Time) (map[string]int, error) {
	rows, err := s.db.Query(
		`SELECT sensor_id, COUNT(*) FROM readings
		 WHERE site_id = ? AND observed_at >= ? AND observed_at <= ? AND quality != ?
		 GROUP BY sensor_id`,
		siteID, formatTime(from), formatTime(to), string(model.QualityRejected))
	if err != nil {
		return nil, wrap("count accepted", err)
	}
	defer rows.Close()
	counts := make(map[string]int)
	for rows.Next() {
		var sensorID string
		var count int
		if err := rows.Scan(&sensorID, &count); err != nil {
			return nil, wrap("scan accepted count", err)
		}
		counts[sensorID] = count
	}
	return counts, rows.Err()
}

func (s *Store) CountSiteReadingsSince(siteID string, since time.Time) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM readings WHERE site_id = ? AND received_at >= ?`,
		siteID, formatTime(since)).Scan(&count)
	if err != nil {
		return 0, wrap("count site readings", err)
	}
	return count, nil
}

func (s *Store) LatestReadingPerType(chamberID string) ([]model.Reading, error) {
	rows, err := s.db.Query(
		`SELECT id, sensor_id, site_id, chamber_id, sensor_type, observed_at,
		        raw_value, value, unit, quality, batch_id, checksum, received_at
		 FROM readings
		 WHERE id IN (
		   SELECT id FROM readings
		   WHERE chamber_id = ?
		   GROUP BY sensor_type
		   HAVING MAX(observed_at)
		 )`, chamberID)
	if err != nil {
		return nil, wrap("latest per type", err)
	}
	defer rows.Close()
	var readings []model.Reading
	for rows.Next() {
		reading, scanErr := scanReading(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		readings = append(readings, reading)
	}
	return readings, rows.Err()
}

const readingColumns = `SELECT id, sensor_id, site_id, chamber_id, sensor_type, observed_at,
	raw_value, value, unit, quality, batch_id, checksum, received_at FROM readings`

const maxReadingLimit = 10000

func scanReading(row rowScanner) (model.Reading, error) {
	var (
		reading   model.Reading
		sensorTyp string
		observed  string
		quality   string
		checksum  int64
		received  string
	)
	if err := row.Scan(&reading.ID, &reading.SensorID, &reading.SiteID, &reading.ChamberID,
		&sensorTyp, &observed, &reading.RawValue, &reading.Value, &reading.Unit,
		&quality, &reading.BatchID, &checksum, &received); err != nil {
		return model.Reading{}, mapNotFound(err)
	}
	reading.SensorType = model.SensorType(sensorTyp)
	reading.ObservedAt = parseTime(observed)
	reading.Quality = model.ReadingQuality(quality)
	reading.Checksum = uint32(checksum)
	reading.ReceivedAt = parseTime(received)
	return reading, nil
}
