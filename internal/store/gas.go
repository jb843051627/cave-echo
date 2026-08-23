package store

import (
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
)

func (s *Store) CreateGasSample(sample model.GasSample) error {
	_, err := s.db.Exec(
		`INSERT INTO gas_samples(id, chamber_id, sampled_at, co2_ppm, oxygen_percent,
		   radon_bqm3, temperature_c, humidity_percent, method, technician, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		sample.ID, sample.ChamberID, formatTime(sample.SampledAt), sample.CO2PPM,
		sample.OxygenPercent, sample.RadonBqM3, sample.TemperatureC, sample.HumidityPercent,
		string(sample.Method), sample.Technician, formatTime(sample.CreatedAt),
	)
	return wrap("create gas sample", err)
}

func (s *Store) ListGasSamples(chamberID string, from, to time.Time, limit int) ([]model.GasSample, error) {
	query := `SELECT id, chamber_id, sampled_at, co2_ppm, oxygen_percent, radon_bqm3,
		temperature_c, humidity_percent, method, technician, created_at FROM gas_samples WHERE 1=1`
	var args []any
	if chamberID != "" {
		query += ` AND chamber_id = ?`
		args = append(args, chamberID)
	}
	if !from.IsZero() {
		query += ` AND sampled_at >= ?`
		args = append(args, formatTime(from))
	}
	if !to.IsZero() {
		query += ` AND sampled_at <= ?`
		args = append(args, formatTime(to))
	}
	if limit <= 0 || limit > 5000 {
		limit = 2000
	}
	query += ` ORDER BY sampled_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, wrap("list gas samples", err)
	}
	defer rows.Close()
	var samples []model.GasSample
	for rows.Next() {
		sample, scanErr := scanGas(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		samples = append(samples, sample)
	}
	return samples, rows.Err()
}

func (s *Store) LatestGasSample(chamberID string) (model.GasSample, error) {
	row := s.db.QueryRow(
		`SELECT id, chamber_id, sampled_at, co2_ppm, oxygen_percent, radon_bqm3,
		   temperature_c, humidity_percent, method, technician, created_at
		 FROM gas_samples WHERE chamber_id = ? ORDER BY sampled_at DESC LIMIT 1`, chamberID)
	return scanGas(row)
}

func scanGas(row rowScanner) (model.GasSample, error) {
	var (
		sample    model.GasSample
		method    string
		sampledAt string
		createdAt string
	)
	if err := row.Scan(&sample.ID, &sample.ChamberID, &sampledAt, &sample.CO2PPM,
		&sample.OxygenPercent, &sample.RadonBqM3, &sample.TemperatureC, &sample.HumidityPercent,
		&method, &sample.Technician, &createdAt); err != nil {
		return model.GasSample{}, mapNotFound(err)
	}
	sample.Method = model.GasMethod(method)
	sample.SampledAt = parseTime(sampledAt)
	sample.CreatedAt = parseTime(createdAt)
	return sample, nil
}
