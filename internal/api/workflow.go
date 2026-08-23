package api

import (
	"net/http"

	"github.com/jb843051627/cave-echo/internal/model"
)

func (s *Server) handleRunAssessment(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ChamberID string `json:"chamber_id"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		s.fail(w, err)
		return
	}
	assessment, err := s.service.RunAssessment(input.ChamberID)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, assessment)
}

func (s *Server) handleListAssessments(w http.ResponseWriter, r *http.Request) {
	assessments, err := s.service.ListAssessments(queryValue(r, "chamber_id"), queryInt(r, "limit", 100))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"assessments": assessments})
}

func (s *Server) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := s.service.ListAlerts(queryValue(r, "site_id"), queryValue(r, "chamber_id"), queryValue(r, "status"), queryInt(r, "limit", 500))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"alerts": alerts})
}

func (s *Server) handleAcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	alert, err := s.service.AcknowledgeAlert(pathValue(r, "alertID"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, alert)
}

func (s *Server) handleCloseAlert(w http.ResponseWriter, r *http.Request) {
	alert, err := s.service.CloseAlert(pathValue(r, "alertID"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, alert)
}

func (s *Server) handleCreateSurvey(w http.ResponseWriter, r *http.Request) {
	var input model.CreateSurveyInput
	if err := decodeJSON(w, r, &input); err != nil {
		s.fail(w, err)
		return
	}
	survey, err := s.service.CreateSurvey(input)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, survey)
}

func (s *Server) handleListSurveys(w http.ResponseWriter, r *http.Request) {
	activeOnly := queryValue(r, "active") == "true"
	surveys, err := s.service.ListSurveys(queryValue(r, "site_id"), activeOnly)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"surveys": surveys})
}

func (s *Server) handleSurveyDetail(w http.ResponseWriter, r *http.Request) {
	detail, err := s.service.SurveyDetail(pathValue(r, "surveyID"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleTransitionSurvey(w http.ResponseWriter, r *http.Request) {
	var input model.TransitionSurveyInput
	if err := decodeJSON(w, r, &input); err != nil {
		s.fail(w, err)
		return
	}
	survey, err := s.service.TransitionSurvey(pathValue(r, "surveyID"), input)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, survey)
}

func (s *Server) handleAddNote(w http.ResponseWriter, r *http.Request) {
	var input model.CreateNoteInput
	if err := decodeJSON(w, r, &input); err != nil {
		s.fail(w, err)
		return
	}
	note, err := s.service.AddNote(input)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, note)
}

func (s *Server) handleListNotes(w http.ResponseWriter, r *http.Request) {
	notes, err := s.service.ListNotes(queryValue(r, "site_id"), queryValue(r, "chamber_id"), queryInt(r, "limit", 500))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"notes": notes})
}
