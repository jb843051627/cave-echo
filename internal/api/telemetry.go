package api

import (
	"net/http"

	"github.com/jb843051627/cave-echo/internal/model"
	"github.com/jb843051627/cave-echo/internal/service"
	"github.com/jb843051627/cave-echo/internal/store"
)

func (s *Server) handleIngestBatch(w http.ResponseWriter, r *http.Request) {
	var batch model.TelemetryBatch
	if err := decodeJSON(w, r, &batch); err != nil {
		s.fail(w, err)
		return
	}
	inserted, err := s.service.IngestBatch(batch)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"accepted": inserted,
		"declared": len(batch.Readings),
	})
}

type checksumRequest struct {
	Readings []model.TelemetryPoint `json:"readings"`
}

func (s *Server) handleExpectedChecksum(w http.ResponseWriter, r *http.Request) {
	var req checksumRequest
	if err := decodeJSON(w, r, &req); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"checksum": service.ExpectedChecksum(req.Readings),
	})
}

func (s *Server) handleListReadings(w http.ResponseWriter, r *http.Request) {
	filter, err := readingFilterFromQuery(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	readings, err := s.service.ListReadings(filter)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"readings": readings})
}

func (s *Server) handleCompleteness(w http.ResponseWriter, r *http.Request) {
	report, err := s.service.SiteCompleteness(queryValue(r, "site_id"), queryWindow(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleReadingsCSV(w http.ResponseWriter, r *http.Request) {
	from, err := queryTime(r, "from")
	if err != nil {
		s.fail(w, service.ErrInvalid)
		return
	}
	to, err := queryTime(r, "to")
	if err != nil {
		s.fail(w, service.ErrInvalid)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=readings-utc.csv")
	if err := s.service.WriteReadingsCSV(w, queryValue(r, "site_id"), from, to); err != nil {
		s.fail(w, err)
	}
}

func (s *Server) handleAssessmentsCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=assessments-utc.csv")
	if err := s.service.WriteAssessmentsCSV(w, queryValue(r, "chamber_id")); err != nil {
		s.fail(w, err)
	}
}

func (s *Server) handleExportBundle(w http.ResponseWriter, r *http.Request) {
	bundle, err := s.service.ExportBundle()
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bundle)
}

func readingFilterFromQuery(r *http.Request) (store.ReadingFilter, error) {
	filter := store.ReadingFilter{
		SensorID:  queryValue(r, "sensor_id"),
		SiteID:    queryValue(r, "site_id"),
		ChamberID: queryValue(r, "chamber_id"),
		Limit:     queryInt(r, "limit", 1000),
	}
	if raw := queryValue(r, "type"); raw != "" {
		kind := model.SensorType(raw)
		if !kind.Valid() {
			return store.ReadingFilter{}, service.ErrInvalid
		}
		filter.Type = kind
	}
	if raw := queryValue(r, "quality"); raw != "" {
		quality := model.ReadingQuality(raw)
		if !quality.Valid() {
			return store.ReadingFilter{}, service.ErrInvalid
		}
		filter.Quality = quality
	}
	from, err := queryTime(r, "from")
	if err != nil {
		return store.ReadingFilter{}, service.ErrInvalid
	}
	to, err := queryTime(r, "to")
	if err != nil {
		return store.ReadingFilter{}, service.ErrInvalid
	}
	if !from.IsZero() && !to.IsZero() && to.Before(from) {
		return store.ReadingFilter{}, service.ErrInvalid
	}
	filter.From = from
	filter.To = to
	return filter, nil
}
