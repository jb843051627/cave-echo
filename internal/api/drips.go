package api

import (
	"net/http"
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
)

func (s *Server) handleCreateDrip(w http.ResponseWriter, r *http.Request) {
	var input model.CreateDripInput
	if err := decodeJSON(w, r, &input); err != nil {
		s.fail(w, err)
		return
	}
	event, err := s.service.RecordDripEvent(input)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, event)
}

func (s *Server) handleListDrips(w http.ResponseWriter, r *http.Request) {
	from, to, ok := parseWindowQuery(w, r)
	if !ok {
		return
	}
	events, err := s.service.ListDripEvents(queryValue(r, "chamber_id"), from, to, queryInt(r, "limit", 500))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"drips": events})
}

func (s *Server) handleDripTrend(w http.ResponseWriter, r *http.Request) {
	trend, err := s.service.DripTrend(queryValue(r, "chamber_id"), queryWindow(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, trend)
}

func (s *Server) handleCreateGasSample(w http.ResponseWriter, r *http.Request) {
	var input model.CreateGasInput
	if err := decodeJSON(w, r, &input); err != nil {
		s.fail(w, err)
		return
	}
	sample, err := s.service.RecordGasSample(input)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sample)
}

func (s *Server) handleListGasSamples(w http.ResponseWriter, r *http.Request) {
	samples, err := s.service.ListGasSamples(queryValue(r, "chamber_id"), queryInt(r, "limit", 200))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"gas_samples": samples})
}

func parseWindowQuery(w http.ResponseWriter, r *http.Request) (time.Time, time.Time, bool) {
	from, err := queryTime(r, "from")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid from timestamp")
		return time.Time{}, time.Time{}, false
	}
	to, err := queryTime(r, "to")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid to timestamp")
		return time.Time{}, time.Time{}, false
	}
	return from, to, true
}
