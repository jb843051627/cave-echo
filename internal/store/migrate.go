package store

import (
	"context"
	"fmt"
)

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS schema_meta (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS cave_sites (
		id TEXT PRIMARY KEY,
		code TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		latitude REAL NOT NULL,
		longitude REAL NOT NULL,
		elevation_m REAL NOT NULL DEFAULT 0,
		protection_grade TEXT NOT NULL,
		status TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_cave_sites_status ON cave_sites(status)`,
	`CREATE TABLE IF NOT EXISTS chambers (
		id TEXT PRIMARY KEY,
		site_id TEXT NOT NULL REFERENCES cave_sites(id),
		name TEXT NOT NULL,
		temperature_band TEXT NOT NULL DEFAULT '',
		airflow_direction TEXT NOT NULL DEFAULT '',
		isolation_boundary TEXT NOT NULL DEFAULT '',
		bat_habitat INTEGER NOT NULL DEFAULT 0,
		protection_rule TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		UNIQUE(site_id, name)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_chambers_site ON chambers(site_id)`,
	`CREATE TABLE IF NOT EXISTS sensors (
		id TEXT PRIMARY KEY,
		site_id TEXT NOT NULL REFERENCES cave_sites(id),
		chamber_id TEXT NOT NULL REFERENCES chambers(id),
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		unit TEXT NOT NULL,
		min_value REAL NOT NULL,
		max_value REAL NOT NULL,
		calibration_offset REAL NOT NULL DEFAULT 0,
		warning_threshold REAL NOT NULL DEFAULT 0,
		critical_threshold REAL NOT NULL DEFAULT 0,
		sample_interval_sec INTEGER NOT NULL DEFAULT 300,
		last_heartbeat_at TEXT NOT NULL DEFAULT '',
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		UNIQUE(chamber_id, type)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_sensors_site ON sensors(site_id)`,
	`CREATE INDEX IF NOT EXISTS idx_sensors_chamber ON sensors(chamber_id)`,
	`CREATE TABLE IF NOT EXISTS readings (
		id TEXT PRIMARY KEY,
		sensor_id TEXT NOT NULL REFERENCES sensors(id),
		site_id TEXT NOT NULL,
		chamber_id TEXT NOT NULL,
		sensor_type TEXT NOT NULL,
		observed_at TEXT NOT NULL,
		raw_value REAL NOT NULL,
		value REAL NOT NULL,
		unit TEXT NOT NULL,
		quality TEXT NOT NULL,
		batch_id TEXT NOT NULL,
		checksum INTEGER NOT NULL DEFAULT 0,
		received_at TEXT NOT NULL
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_readings_sensor_time ON readings(sensor_id, observed_at)`,
	`CREATE INDEX IF NOT EXISTS idx_readings_site_time ON readings(site_id, observed_at)`,
	`CREATE INDEX IF NOT EXISTS idx_readings_chamber ON readings(chamber_id, observed_at)`,
	`CREATE TABLE IF NOT EXISTS drip_events (
		id TEXT PRIMARY KEY,
		chamber_id TEXT NOT NULL REFERENCES chambers(id),
		observed_at TEXT NOT NULL,
		rate_per_minute REAL NOT NULL,
		mineralization REAL NOT NULL DEFAULT 0,
		color TEXT NOT NULL,
		location TEXT NOT NULL DEFAULT '',
		duration_seconds INTEGER NOT NULL DEFAULT 0,
		observer TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_drip_chamber_time ON drip_events(chamber_id, observed_at)`,
	`CREATE TABLE IF NOT EXISTS gas_samples (
		id TEXT PRIMARY KEY,
		chamber_id TEXT NOT NULL REFERENCES chambers(id),
		sampled_at TEXT NOT NULL,
		co2_ppm REAL NOT NULL,
		oxygen_percent REAL NOT NULL,
		radon_bqm3 REAL NOT NULL DEFAULT 0,
		temperature_c REAL NOT NULL DEFAULT 0,
		humidity_percent REAL NOT NULL DEFAULT 0,
		method TEXT NOT NULL,
		technician TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_gas_chamber_time ON gas_samples(chamber_id, sampled_at)`,
	`CREATE TABLE IF NOT EXISTS surveys (
		id TEXT PRIMARY KEY,
		site_id TEXT NOT NULL REFERENCES cave_sites(id),
		chamber_id TEXT NOT NULL REFERENCES chambers(id),
		transect TEXT NOT NULL,
		surface_condition TEXT NOT NULL DEFAULT '',
		crystal_change_mm REAL NOT NULL DEFAULT 0,
		stage TEXT NOT NULL,
		started_at TEXT NOT NULL,
		completed_at TEXT NOT NULL DEFAULT '',
		findings TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_surveys_site_stage ON surveys(site_id, stage)`,
	`CREATE INDEX IF NOT EXISTS idx_surveys_chamber ON surveys(chamber_id)`,
	`CREATE TABLE IF NOT EXISTS stability_assessments (
		id TEXT PRIMARY KEY,
		site_id TEXT NOT NULL,
		chamber_id TEXT NOT NULL,
		assessed_at TEXT NOT NULL,
		score REAL NOT NULL,
		condensation_risk REAL NOT NULL,
		gas_risk REAL NOT NULL,
		drip_risk REAL NOT NULL,
		airflow_risk REAL NOT NULL,
		completeness REAL NOT NULL,
		level TEXT NOT NULL,
		summary TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_assessments_chamber ON stability_assessments(chamber_id, assessed_at)`,
	`CREATE INDEX IF NOT EXISTS idx_assessments_site ON stability_assessments(site_id, assessed_at)`,
	`CREATE TABLE IF NOT EXISTS alerts (
		id TEXT PRIMARY KEY,
		site_id TEXT NOT NULL,
		chamber_id TEXT NOT NULL,
		sensor_id TEXT NOT NULL DEFAULT '',
		kind TEXT NOT NULL,
		severity TEXT NOT NULL,
		status TEXT NOT NULL,
		dedup_key TEXT NOT NULL,
		message TEXT NOT NULL,
		first_seen_at TEXT NOT NULL,
		last_seen_at TEXT NOT NULL,
		acknowledged_at TEXT NOT NULL DEFAULT '',
		closed_at TEXT NOT NULL DEFAULT '',
		occurrences INTEGER NOT NULL DEFAULT 1
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_alerts_dedup_active ON alerts(dedup_key, status) WHERE status != 'closed'`,
	`CREATE INDEX IF NOT EXISTS idx_alerts_site_status ON alerts(site_id, status)`,
	`CREATE INDEX IF NOT EXISTS idx_alerts_chamber ON alerts(chamber_id)`,
	`CREATE TABLE IF NOT EXISTS conservation_notes (
		id TEXT PRIMARY KEY,
		site_id TEXT NOT NULL,
		chamber_id TEXT NOT NULL,
		survey_id TEXT NOT NULL DEFAULT '',
		author TEXT NOT NULL,
		category TEXT NOT NULL,
		note TEXT NOT NULL,
		action_outcome TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_notes_chamber ON conservation_notes(chamber_id, created_at)`,
	`CREATE INDEX IF NOT EXISTS idx_notes_site ON conservation_notes(site_id)`,
}

func (s *Store) Migrate() error {
	ctx := context.Background()
	// Bootstrap the version table itself, then read the current version.
	if _, err := s.db.ExecContext(ctx, migrations[0]); err != nil {
		return fmt.Errorf("store: bootstrap schema_meta: %w", err)
	}
	var current int
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0) FROM schema_meta`).Scan(&current); err != nil {
		return fmt.Errorf("store: read schema version: %w", err)
	}
	for i, statement := range migrations {
		version := i + 1
		if i == 0 || version <= current {
			continue
		}
		err := s.InTx(func(tx *sqlTx) error {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, `INSERT INTO schema_meta(version, applied_at) VALUES(?, datetime('now'))`, version)
			return err
		})
		if err != nil {
			return fmt.Errorf("store: apply migration %d: %w", version, err)
		}
	}
	return nil
}
