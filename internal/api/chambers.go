package api

import (
	"net/http"

	"github.com/jb843051627/cave-echo/internal/model"
)

func (s *Server) handleCreateChamber(w http.ResponseWriter, r *http.Request) {
	var input model.CreateChamberInput
	if err := decodeJSON(w, r, &input); err != nil {
		s.fail(w, err)
		return
	}
	chamber, err := s.service.CreateChamber(pathValue(r, "siteID"), input)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, chamber)
}

func (s *Server) handleListChambers(w http.ResponseWriter, r *http.Request) {
	chambers, err := s.service.ListChambers(pathValue(r, "siteID"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"chambers": chambers})
}

func (s *Server) handleChamberSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.service.ChamberSnapshot(pathValue(r, "chamberID"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleRegisterSensor(w http.ResponseWriter, r *http.Request) {
	var input model.CreateSensorInput
	if err := decodeJSON(w, r, &input); err != nil {
		s.fail(w, err)
		return
	}
	sensor, err := s.service.RegisterSensor(input)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sensor)
}

func (s *Server) handleListSensors(w http.ResponseWriter, r *http.Request) {
	sensors, err := s.service.ListSensors(queryValue(r, "site_id"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sensors": sensors})
}

func (s *Server) handleGetSensor(w http.ResponseWriter, r *http.Request) {
	sensor, err := s.service.GetSensor(pathValue(r, "sensorID"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sensor)
}

type enabledInput struct {
	Enabled bool `json:"enabled"`
}

func (s *Server) handleSetSensorEnabled(w http.ResponseWriter, r *http.Request) {
	var input enabledInput
	if err := decodeJSON(w, r, &input); err != nil {
		s.fail(w, err)
		return
	}
	sensor, err := s.service.SetSensorEnabled(pathValue(r, "sensorID"), input.Enabled)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sensor)
}

type thresholdsInput struct {
	WarningThreshold  float64 `json:"warning_threshold"`
	CriticalThreshold float64 `json:"critical_threshold"`
}

func (s *Server) handleUpdateThresholds(w http.ResponseWriter, r *http.Request) {
	var input thresholdsInput
	if err := decodeJSON(w, r, &input); err != nil {
		s.fail(w, err)
		return
	}
	sensor, err := s.service.UpdateSensorThresholds(pathValue(r, "sensorID"), input.WarningThreshold, input.CriticalThreshold)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sensor)
}
