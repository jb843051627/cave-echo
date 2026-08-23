package store

import (
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
)

func (s *Store) CreateSensor(sensor model.Sensor) error {
	_, err := s.db.Exec(
		`INSERT INTO sensors(id, site_id, chamber_id, name, type, unit, min_value, max_value,
		   calibration_offset, warning_threshold, critical_threshold, sample_interval_sec,
		   last_heartbeat_at, enabled, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		sensor.ID, sensor.SiteID, sensor.ChamberID, sensor.Name, string(sensor.Type), sensor.Unit,
		sensor.MinValue, sensor.MaxValue, sensor.CalibrationOffset,
		sensor.WarningThreshold, sensor.CriticalThreshold, sensor.SampleIntervalSec,
		formatTime(sensor.LastHeartbeatAt), boolToInt(sensor.Enabled),
		formatTime(sensor.CreatedAt), formatTime(sensor.UpdatedAt),
	)
	if isUniqueViolation(err) {
		return wrap("create sensor", ErrDuplicateSensor)
	}
	return wrap("create sensor", err)
}

func (s *Store) GetSensor(id string) (model.Sensor, error) {
	row := s.db.QueryRow(sensorColumns+` FROM sensors WHERE id = ?`, id)
	return scanSensor(row)
}

func (s *Store) ListSensorsBySite(siteID string) ([]model.Sensor, error) {
	return s.querySensors(sensorColumns+` FROM sensors WHERE site_id = ? ORDER BY name`, siteID)
}

func (s *Store) ListSensorsByChamber(chamberID string) ([]model.Sensor, error) {
	return s.querySensors(sensorColumns+` FROM sensors WHERE chamber_id = ? ORDER BY type`, chamberID)
}

func (s *Store) ListEnabledSensors() ([]model.Sensor, error) {
	return s.querySensors(sensorColumns + ` FROM sensors WHERE enabled = 1 ORDER BY site_id, name`)
}

func (s *Store) SetSensorEnabled(id string, enabled bool, now time.Time) error {
	res, err := s.db.Exec(`UPDATE sensors SET enabled = ?, updated_at = ? WHERE id = ?`,
		boolToInt(enabled), formatTime(now), id)
	if err != nil {
		return wrap("set sensor enabled", err)
	}
	return requireAffected(res, "sensor")
}

func (s *Store) UpdateThresholds(id string, warning, critical float64, now time.Time) error {
	res, err := s.db.Exec(
		`UPDATE sensors SET warning_threshold = ?, critical_threshold = ?, updated_at = ? WHERE id = ?`,
		warning, critical, formatTime(now), id)
	if err != nil {
		return wrap("update thresholds", err)
	}
	return requireAffected(res, "sensor")
}

func (s *Store) TouchHeartbeat(id string, at time.Time) error {
	res, err := s.db.Exec(`UPDATE sensors SET last_heartbeat_at = ?, updated_at = ? WHERE id = ?`,
		formatTime(at), formatTime(at), id)
	if err != nil {
		return wrap("touch heartbeat", err)
	}
	return requireAffected(res, "sensor")
}

const sensorColumns = `SELECT id, site_id, chamber_id, name, type, unit, min_value, max_value,
	calibration_offset, warning_threshold, critical_threshold, sample_interval_sec,
	last_heartbeat_at, enabled, created_at, updated_at`

func (s *Store) querySensors(query string, args ...any) ([]model.Sensor, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, wrap("query sensors", err)
	}
	defer rows.Close()
	var sensors []model.Sensor
	for rows.Next() {
		sensor, err := scanSensor(rows)
		if err != nil {
			return nil, err
		}
		sensors = append(sensors, sensor)
	}
	return sensors, rows.Err()
}

func scanSensor(row rowScanner) (model.Sensor, error) {
	var (
		sensor    model.Sensor
		sType     string
		heartbeat string
		enabled   int
		created   string
		updated   string
	)
	if err := row.Scan(&sensor.ID, &sensor.SiteID, &sensor.ChamberID, &sensor.Name, &sType,
		&sensor.Unit, &sensor.MinValue, &sensor.MaxValue, &sensor.CalibrationOffset,
		&sensor.WarningThreshold, &sensor.CriticalThreshold, &sensor.SampleIntervalSec,
		&heartbeat, &enabled, &created, &updated); err != nil {
		return model.Sensor{}, mapNotFound(err)
	}
	sensor.Type = model.SensorType(sType)
	sensor.LastHeartbeatAt = parseTime(heartbeat)
	sensor.Enabled = enabled == 1
	sensor.CreatedAt = parseTime(created)
	sensor.UpdatedAt = parseTime(updated)
	return sensor, nil
}
