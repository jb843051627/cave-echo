package service

import (
	"fmt"
	"strings"

	"github.com/jb843051627/cave-echo/internal/model"
	"github.com/jb843051627/cave-echo/internal/validation"
)

// AddNote records a conservation note; it can reference a survey and, when
// attached to an active survey, is validated against that survey's site.
func (s *Service) AddNote(input model.CreateNoteInput) (model.ConservationNote, error) {
	site, err := s.requireSite(input.SiteID)
	if err != nil {
		return model.ConservationNote{}, err
	}
	chamber, err := s.requireChamber(input.ChamberID)
	if err != nil {
		return model.ConservationNote{}, err
	}
	if chamber.SiteID != site.ID {
		return model.ConservationNote{}, fmt.Errorf("%w: chamber does not belong to site", ErrInvalid)
	}
	if !input.Category.Valid() {
		return model.ConservationNote{}, fmt.Errorf("%w: unknown note category %q", ErrInvalid, input.Category)
	}
	if err := validation.RequireText("note", input.Note, 4000); err != nil {
		return model.ConservationNote{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	note := model.ConservationNote{
		ID:            model.NewID("note"),
		SiteID:        site.ID,
		ChamberID:     chamber.ID,
		SurveyID:      strings.TrimSpace(input.SurveyID),
		Author:        strings.TrimSpace(input.Author),
		Category:      input.Category,
		Note:          strings.TrimSpace(input.Note),
		ActionOutcome: strings.TrimSpace(input.ActionOutcome),
		CreatedAt:     s.now(),
	}
	if note.Author == "" {
		note.Author = "unknown"
	}
	if note.SurveyID != "" {
		survey, err := s.store.GetSurvey(note.SurveyID)
		if err != nil {
			return model.ConservationNote{}, fmt.Errorf("%w: referenced survey missing", ErrInvalid)
		}
		if survey.SiteID != site.ID {
			return model.ConservationNote{}, fmt.Errorf("%w: survey belongs to another site", ErrInvalid)
		}
	}
	if err := s.store.CreateNote(note); err != nil {
		return model.ConservationNote{}, err
	}
	s.bump("notes_added")
	return note, nil
}

func (s *Service) ListNotes(siteID, chamberID string, limit int) ([]model.ConservationNote, error) {
	if siteID == "" && chamberID == "" {
		return nil, fmt.Errorf("%w: site or chamber filter required", ErrInvalid)
	}
	return s.store.ListNotes(siteID, chamberID, limit)
}

func (s *Service) ListSurveyNotes(surveyID string) ([]model.ConservationNote, error) {
	if !model.IsID(surveyID) {
		return nil, fmt.Errorf("%w: survey id required", ErrInvalid)
	}
	return s.store.ListNotesBySurvey(surveyID)
}
