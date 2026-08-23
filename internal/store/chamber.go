package store

import (
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
)

func (s *Store) CreateChamber(chamber model.Chamber) error {
	_, err := s.db.Exec(
		`INSERT INTO chambers(id, site_id, name, temperature_band, airflow_direction, isolation_boundary, bat_habitat, protection_rule, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?)`,
		chamber.ID, chamber.SiteID, chamber.Name, chamber.TemperatureBand, chamber.AirflowDirection,
		chamber.IsolationBoundary, boolToInt(chamber.BatHabitat), chamber.ProtectionRule,
		formatTime(chamber.CreatedAt), formatTime(chamber.UpdatedAt),
	)
	if isUniqueViolation(err) {
		return wrap("create chamber", ErrDuplicateChamber)
	}
	return wrap("create chamber", err)
}

func (s *Store) GetChamber(id string) (model.Chamber, error) {
	row := s.db.QueryRow(chamberColumns+` FROM chambers WHERE id = ?`, id)
	return scanChamber(row)
}

func (s *Store) ListChambersBySite(siteID string) ([]model.Chamber, error) {
	rows, err := s.db.Query(chamberColumns+` FROM chambers WHERE site_id = ? ORDER BY name`, siteID)
	if err != nil {
		return nil, wrap("list chambers", err)
	}
	defer rows.Close()
	var chambers []model.Chamber
	for rows.Next() {
		chamber, err := scanChamber(rows)
		if err != nil {
			return nil, err
		}
		chambers = append(chambers, chamber)
	}
	return chambers, rows.Err()
}

func (s *Store) ListAllChambers() ([]model.Chamber, error) {
	rows, err := s.db.Query(chamberColumns + ` FROM chambers ORDER BY site_id, name`)
	if err != nil {
		return nil, wrap("list all chambers", err)
	}
	defer rows.Close()
	var chambers []model.Chamber
	for rows.Next() {
		chamber, err := scanChamber(rows)
		if err != nil {
			return nil, err
		}
		chambers = append(chambers, chamber)
	}
	return chambers, rows.Err()
}

func (s *Store) UpdateChamberProtection(id, rule string, now time.Time) error {
	res, err := s.db.Exec(`UPDATE chambers SET protection_rule = ?, updated_at = ? WHERE id = ?`,
		rule, formatTime(now), id)
	if err != nil {
		return wrap("update chamber protection", err)
	}
	return requireAffected(res, "chamber")
}

const chamberColumns = `SELECT id, site_id, name, temperature_band, airflow_direction, isolation_boundary, bat_habitat, protection_rule, created_at, updated_at`

func scanChamber(row rowScanner) (model.Chamber, error) {
	var (
		chamber    model.Chamber
		batHabitat int
		created    string
		updated    string
	)
	if err := row.Scan(&chamber.ID, &chamber.SiteID, &chamber.Name, &chamber.TemperatureBand,
		&chamber.AirflowDirection, &chamber.IsolationBoundary, &batHabitat, &chamber.ProtectionRule,
		&created, &updated); err != nil {
		return model.Chamber{}, mapNotFound(err)
	}
	chamber.BatHabitat = batHabitat == 1
	chamber.CreatedAt = parseTime(created)
	chamber.UpdatedAt = parseTime(updated)
	return chamber, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
