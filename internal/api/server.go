package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/jb843051627/cave-echo/internal/metrics"
	"github.com/jb843051627/cave-echo/internal/service"
)

type Dependencies struct {
	Service *service.Service
	Metrics *metrics.Registry
	Static  string
}

type Server struct {
	service *service.Service
	metrics *metrics.Registry
	static  string
}

func New(deps Dependencies) *Server {
	return &Server{
		service: deps.Service,
		metrics: deps.Metrics,
		static:  deps.Static,
	}
}

// Handler builds the full HTTP mux: health probe, API routes and the static console.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /metrics.json", s.handleMetrics)

	// overview & exports
	mux.HandleFunc("GET /api/overview", s.handleOverview)
	mux.HandleFunc("GET /api/export/readings.csv", s.handleReadingsCSV)
	mux.HandleFunc("GET /api/export/assessments.csv", s.handleAssessmentsCSV)
	mux.HandleFunc("GET /api/export/bundle.json", s.handleExportBundle)
	mux.HandleFunc("GET /api/completeness", s.handleCompleteness)

	// sites
	mux.HandleFunc("POST /api/sites", s.handleCreateSite)
	mux.HandleFunc("GET /api/sites", s.handleListSites)
	mux.HandleFunc("GET /api/sites/{siteID}", s.handleGetSite)
	mux.HandleFunc("PATCH /api/sites/{siteID}/status", s.handleUpdateSiteStatus)

	// chambers
	mux.HandleFunc("POST /api/sites/{siteID}/chambers", s.handleCreateChamber)
	mux.HandleFunc("GET /api/sites/{siteID}/chambers", s.handleListChambers)
	mux.HandleFunc("GET /api/chambers/{chamberID}", s.handleChamberSnapshot)

	// sensors
	mux.HandleFunc("POST /api/sensors", s.handleRegisterSensor)
	mux.HandleFunc("GET /api/sensors", s.handleListSensors)
	mux.HandleFunc("GET /api/sensors/{sensorID}", s.handleGetSensor)
	mux.HandleFunc("PATCH /api/sensors/{sensorID}/enabled", s.handleSetSensorEnabled)
	mux.HandleFunc("PATCH /api/sensors/{sensorID}/thresholds", s.handleUpdateThresholds)

	// telemetry
	mux.HandleFunc("POST /api/telemetry/batches", s.handleIngestBatch)
	mux.HandleFunc("POST /api/telemetry/checksum", s.handleExpectedChecksum)
	mux.HandleFunc("GET /api/readings", s.handleListReadings)

	// drips & gas
	mux.HandleFunc("POST /api/drips", s.handleCreateDrip)
	mux.HandleFunc("GET /api/drips", s.handleListDrips)
	mux.HandleFunc("GET /api/drips/trend", s.handleDripTrend)
	mux.HandleFunc("POST /api/gas-samples", s.handleCreateGasSample)
	mux.HandleFunc("GET /api/gas-samples", s.handleListGasSamples)

	// assessments
	mux.HandleFunc("POST /api/assessments", s.handleRunAssessment)
	mux.HandleFunc("GET /api/assessments", s.handleListAssessments)

	// alerts
	mux.HandleFunc("GET /api/alerts", s.handleListAlerts)
	mux.HandleFunc("POST /api/alerts/{alertID}/acknowledge", s.handleAcknowledgeAlert)
	mux.HandleFunc("POST /api/alerts/{alertID}/close", s.handleCloseAlert)

	// surveys & notes
	mux.HandleFunc("POST /api/surveys", s.handleCreateSurvey)
	mux.HandleFunc("GET /api/surveys", s.handleListSurveys)
	mux.HandleFunc("GET /api/surveys/{surveyID}", s.handleSurveyDetail)
	mux.HandleFunc("POST /api/surveys/{surveyID}/transition", s.handleTransitionSurvey)
	mux.HandleFunc("POST /api/notes", s.handleAddNote)
	mux.HandleFunc("GET /api/notes", s.handleListNotes)

	mux.Handle("/", s.staticHandler())

	return s.wrap(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":            "ok",
		"service":           "cave-echo",
		"readings_ingested": s.counter("readings_ingested"),
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"counters": s.snapshot()})
}

func (s *Server) counter(key string) int64 {
	if s.metrics == nil {
		return 0
	}
	return s.metrics.Get(key)
}

func (s *Server) snapshot() map[string]int64 {
	if s.metrics == nil {
		return map[string]int64{}
	}
	return s.metrics.Snapshot()
}

// wrap adds recovery and request logging around every handler.
func (s *Server) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("api: panic on %s %s: %v", r.Method, r.URL.Path, rec)
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("api: encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrInvalid),
		errors.Is(err, service.ErrDuplicate),
		errors.Is(err, service.ErrIllegalStage),
		errors.Is(err, service.ErrSeasonBlocked):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("api: internal error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
