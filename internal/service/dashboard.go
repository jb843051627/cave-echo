package service

import (
	"encoding/csv"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
	"github.com/jb843051627/cave-echo/internal/store"
)

// Overview builds the console landing data: per-site summaries, alert
// counters and seasonal notices.
func (s *Service) Overview() (model.Overview, error) {
	now := s.now()
	sites, err := s.store.ListSites()
	if err != nil {
		return model.Overview{}, err
	}
	chambers, err := s.store.ListAllChambers()
	if err != nil {
		return model.Overview{}, err
	}
	chambersBySite := make(map[string][]model.Chamber)
	for _, chamber := range chambers {
		chambersBySite[chamber.SiteID] = append(chambersBySite[chamber.SiteID], chamber)
	}

	overview := model.Overview{
		GeneratedAt: now,
		Sites:       make([]model.SiteSummary, 0, len(sites)),
	}
	windowStart := now.Add(-24 * time.Hour)
	for _, site := range sites {
		sensors, err := s.store.ListSensorsBySite(site.ID)
		if err != nil {
			return model.Overview{}, err
		}
		activeAlerts, criticalAlerts, err := s.store.CountActiveAlerts(site.ID)
		if err != nil {
			return model.Overview{}, err
		}
		if activeAlerts > 0 {
			overview.OpenAlerts += activeAlerts
			overview.CriticalAlerts += criticalAlerts
		}
		completeness, latestReading, err := s.siteCompleteness(site.ID, sensors, windowStart, now)
		if err != nil {
			return model.Overview{}, err
		}
		readingsToday, err := s.store.CountSiteReadingsSince(site.ID, windowStart)
		if err != nil {
			return model.Overview{}, err
		}
		overview.ReadingsToday += readingsToday

		protection := ""
		for _, chamber := range chambersBySite[site.ID] {
			if msg := s.engine.ProtectionMessage(chamber, now); msg != "" {
				protection = msg
				break
			}
		}
		overview.Sites = append(overview.Sites, model.SiteSummary{
			Site:              site,
			ChamberCount:      len(chambersBySite[site.ID]),
			SensorCount:       len(sensors),
			ActiveAlertCount:  activeAlerts,
			Completeness:      completeness,
			LastReadingAt:     latestReading,
			ProtectionMessage: protection,
		})
	}
	overview.SeasonalNotices = s.engine.SeasonalNotices(chambers, now)
	overview.Observability = round2(1 - float64(overview.CriticalAlerts)/float64(overview.OpenAlerts))
	return overview, nil
}

func round2(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

// ExportBundle gathers every entity for the UTC JSON export.
func (s *Service) ExportBundle() (model.ExportBundle, error) {
	bundle := model.ExportBundle{GeneratedAt: s.now()}
	var err error
	if bundle.Sites, err = s.store.ListSites(); err != nil {
		return model.ExportBundle{}, err
	}
	if bundle.Chambers, err = s.store.ListAllChambers(); err != nil {
		return model.ExportBundle{}, err
	}
	for _, site := range bundle.Sites {
		sensors, sensorErr := s.store.ListSensorsBySite(site.ID)
		if sensorErr != nil {
			return model.ExportBundle{}, sensorErr
		}
		bundle.Sensors = append(bundle.Sensors, sensors...)
		to := s.now()
		from := to.Add(-365 * 24 * time.Hour)
		readings, readingErr := s.store.ListReadings(store.ReadingFilter{SiteID: site.ID, From: from, To: to, Limit: 10000})
		if readingErr != nil {
			return model.ExportBundle{}, readingErr
		}
		bundle.Readings = append(bundle.Readings, readings...)
	}
	for _, chamber := range bundle.Chambers {
		drips, dripErr := s.store.ListDripEvents(chamber.ID, time.Time{}, time.Time{}, 5000)
		if dripErr != nil {
			return model.ExportBundle{}, dripErr
		}
		gas, gasErr := s.store.ListGasSamples(chamber.ID, time.Time{}, time.Time{}, 5000)
		if gasErr != nil {
			return model.ExportBundle{}, gasErr
		}
		bundle.Drips = append(bundle.Drips, drips...)
		bundle.GasSamples = append(bundle.GasSamples, gas...)
	}
	for _, site := range bundle.Sites {
		surveys, surveyErr := s.store.ListSurveys(site.ID, false)
		if surveyErr != nil {
			return model.ExportBundle{}, surveyErr
		}
		bundle.Surveys = append(bundle.Surveys, surveys...)
		alerts, alertErr := s.store.ListAlerts(site.ID, "", "", 2000)
		if alertErr != nil {
			return model.ExportBundle{}, alertErr
		}
		bundle.Alerts = append(bundle.Alerts, alerts...)
		assessments, assessErr := s.store.ListAssessments("", site.ID, 2000)
		if assessErr != nil {
			return model.ExportBundle{}, assessErr
		}
		bundle.Assessments = append(bundle.Assessments, assessments...)
		notes, noteErr := s.store.ListNotes(site.ID, "", 2000)
		if noteErr != nil {
			return model.ExportBundle{}, noteErr
		}
		bundle.Notes = append(bundle.Notes, notes...)
	}
	return bundle, nil
}

// WriteReadingsCSV streams a UTC-timestamped readings export.
func (s *Service) WriteReadingsCSV(w io.Writer, siteID string, from, to time.Time) error {
	if _, err := s.requireSite(siteID); err != nil {
		return err
	}
	if to.IsZero() {
		to = s.now()
	}
	if from.IsZero() {
		from = to.Add(-7 * 24 * time.Hour)
	}
	readings, err := s.store.ListReadings(store.ReadingFilter{SiteID: siteID, From: from, To: to})
	if err != nil {
		return err
	}
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{
		"reading_id", "sensor_id", "sensor_type", "site_id", "chamber_id",
		"observed_at_utc", "raw_value", "value", "unit", "quality", "batch_id",
	}); err != nil {
		return err
	}
	for _, reading := range readings {
		row := []string{
			reading.ID,
			reading.SensorID,
			string(reading.SensorType),
			reading.SiteID,
			reading.ChamberID,
			reading.ObservedAt.UTC().Format(time.RFC3339),
			strconv.FormatFloat(reading.RawValue, 'f', -1, 64),
			strconv.FormatFloat(reading.Value, 'f', -1, 64),
			reading.Unit,
			string(reading.Quality),
			reading.BatchID,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

// WriteAssessmentsCSV streams assessment history as UTC CSV.
func (s *Service) WriteAssessmentsCSV(w io.Writer, chamberID string) error {
	if _, err := s.requireChamber(chamberID); err != nil {
		return err
	}
	assessments, err := s.store.ListAssessments(chamberID, "", 1000)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(w)
	header := []string{
		"assessment_id", "chamber_id", "assessed_at_utc", "score", "level",
		"condensation_risk", "gas_risk", "drip_risk", "airflow_risk", "completeness",
	}
	if err := writer.Write(header); err != nil {
		return err
	}
	for _, assessment := range assessments {
		row := []string{
			assessment.ID,
			assessment.ChamberID,
			assessment.AssessedAt.UTC().Format(time.RFC3339),
			formatFloat(assessment.Score),
			string(assessment.Level),
			formatFloat(assessment.CondensationRisk),
			formatFloat(assessment.GasRisk),
			formatFloat(assessment.DripRisk),
			formatFloat(assessment.AirflowRisk),
			formatFloat(assessment.Completeness),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func formatFloat(value float64) string {
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(value, 'f', 4, 64), "0"), ".")
}
