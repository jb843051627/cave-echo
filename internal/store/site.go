package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
)

func (s *Store) CreateSite(site model.CaveSite) error {
	_, err := s.db.Exec(
		`INSERT INTO cave_sites(id, code, name, latitude, longitude, elevation_m, protection_grade, status, description, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		site.ID, site.Code, site.Name, site.Latitude, site.Longitude, site.ElevationM,
		string(site.ProtectionGrade), string(site.Status), site.Description,
		formatTime(site.CreatedAt), formatTime(site.UpdatedAt),
	)
	if isUniqueViolation(err) {
		return fmt.Errorf("store: site code %q already exists", site.Code)
	}
	return wrap("create site", err)
}

func (s *Store) GetSite(id string) (model.CaveSite, error) {
	row := s.db.QueryRow(
		`SELECT id, code, name, latitude, longitude, elevation_m, protection_grade, status, description, created_at, updated_at
		 FROM cave_sites WHERE id = ?`, id)
	return scanSite(row)
}

func (s *Store) GetSiteByCode(code string) (model.CaveSite, error) {
	row := s.db.QueryRow(
		`SELECT id, code, name, latitude, longitude, elevation_m, protection_grade, status, description, created_at, updated_at
		 FROM cave_sites WHERE code = ?`, strings.TrimSpace(code))
	return scanSite(row)
}

func (s *Store) ListSites() ([]model.CaveSite, error) {
	rows, err := s.db.Query(
		`SELECT id, code, name, latitude, longitude, elevation_m, protection_grade, status, description, created_at, updated_at
		 FROM cave_sites ORDER BY code`)
	if err != nil {
		return nil, wrap("list sites", err)
	}
	defer rows.Close()
	var sites []model.CaveSite
	for rows.Next() {
		site, err := scanSite(rows)
		if err != nil {
			return nil, err
		}
		sites = append(sites, site)
	}
	return sites, rows.Err()
}

func (s *Store) UpdateSiteStatus(id string, status model.SiteStatus, now time.Time) error {
	res, err := s.db.Exec(`UPDATE cave_sites SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), formatTime(now), id)
	if err != nil {
		return wrap("update site status", err)
	}
	return requireAffected(res, "site")
}

func (s *Store) UpdateSiteDescription(id, description string, now time.Time) error {
	res, err := s.db.Exec(`UPDATE cave_sites SET description = ?, updated_at = ? WHERE id = ?`,
		description, formatTime(now), id)
	if err != nil {
		return wrap("update site description", err)
	}
	return requireAffected(res, "site")
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSite(row rowScanner) (model.CaveSite, error) {
	var (
		site    model.CaveSite
		grade   string
		status  string
		created string
		updated string
	)
	if err := row.Scan(&site.ID, &site.Code, &site.Name, &site.Latitude, &site.Longitude,
		&site.ElevationM, &grade, &status, &site.Description, &created, &updated); err != nil {
		return model.CaveSite{}, mapNotFound(err)
	}
	site.ProtectionGrade = model.ProtectionGrade(grade)
	site.Status = model.SiteStatus(status)
	site.CreatedAt = parseTime(created)
	site.UpdatedAt = parseTime(updated)
	return site, nil
}
