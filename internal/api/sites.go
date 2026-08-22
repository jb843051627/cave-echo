package api

import (
	"net/http"

	"github.com/jb843051627/cave-echo/internal/model"
)

func (s *Server) handleCreateSite(w http.ResponseWriter, r *http.Request) {
	var input model.CreateSiteInput
	if err := decodeJSON(w, r, &input); err != nil {
		s.fail(w, err)
		return
	}
	site, err := s.service.CreateSite(input)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, site)
}

func (s *Server) handleListSites(w http.ResponseWriter, r *http.Request) {
	sites, err := s.service.ListSites()
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sites": sites})
}

func (s *Server) handleGetSite(w http.ResponseWriter, r *http.Request) {
	summary, err := s.service.SiteSummary(pathValue(r, "siteID"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleUpdateSiteStatus(w http.ResponseWriter, r *http.Request) {
	var input model.UpdateSiteStatusInput
	if err := decodeJSON(w, r, &input); err != nil {
		s.fail(w, err)
		return
	}
	site, err := s.service.UpdateSiteStatus(pathValue(r, "siteID"), input.Status)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, site)
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := s.service.Overview()
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, overview)
}
