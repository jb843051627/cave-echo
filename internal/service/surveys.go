package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/jb843051627/cave-echo/internal/model"
	"github.com/jb843051627/cave-echo/internal/validation"
)

// CreateSurvey opens a protection survey in the planned stage.
func (s *Service) CreateSurvey(input model.CreateSurveyInput) (model.Survey, error) {
	site, err := s.requireSite(input.SiteID)
	if err != nil {
		return model.Survey{}, err
	}
	chamber, err := s.requireChamber(input.ChamberID)
	if err != nil {
		return model.Survey{}, err
	}
	if chamber.SiteID != site.ID {
		return model.Survey{}, fmt.Errorf("%w: chamber does not belong to site", ErrInvalid)
	}
	if err := validation.RequireText("transect", input.Transect, 80); err != nil {
		return model.Survey{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	now := s.now()
	survey := model.Survey{
		ID:               model.NewID("srv"),
		SiteID:           site.ID,
		ChamberID:        chamber.ID,
		Transect:         strings.TrimSpace(input.Transect),
		SurfaceCondition: strings.TrimSpace(input.SurfaceCondition),
		CrystalChangeMM:  input.CrystalChangeMM,
		Stage:            model.SurveyPlanned,
		StartedAt:        now,
		Findings:         strings.TrimSpace(input.Findings),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.store.CreateSurvey(survey); err != nil {
		return model.Survey{}, err
	}
	s.bump("surveys_created")
	return survey, nil
}

func (s *Service) GetSurvey(surveyID string) (model.Survey, error) {
	if !model.IsID(surveyID) {
		return model.Survey{}, fmt.Errorf("%w: survey id required", ErrInvalid)
	}
	return s.store.GetSurvey(surveyID)
}

func (s *Service) ListSurveys(siteID string, activeOnly bool) ([]model.Survey, error) {
	if siteID == "" {
		return nil, fmt.Errorf("%w: site id required", ErrInvalid)
	}
	if _, err := s.requireSite(siteID); err != nil {
		return nil, err
	}
	return s.store.ListSurveys(siteID, activeOnly)
}

// TransitionSurvey advances the draft→fieldwork→review→published style state
// machine. The store enforces compare-and-swap on the previous stage so two
// concurrent transitions cannot both succeed.
func (s *Service) TransitionSurvey(surveyID string, input model.TransitionSurveyInput) (model.Survey, error) {
	survey, err := s.GetSurvey(surveyID)
	if err != nil {
		return model.Survey{}, err
	}
	next := input.Stage
	if !next.Valid() {
		return model.Survey{}, fmt.Errorf("%w: unknown stage %q", ErrInvalid, next)
	}
	if !survey.Stage.CanMoveTo(next) {
		return model.Survey{}, fmt.Errorf("%w: %s -> %s", ErrIllegalStage, survey.Stage, next)
	}
	chamber, err := s.requireChamber(survey.ChamberID)
	if err != nil {
		return model.Survey{}, err
	}
	now := s.now()
	if err := s.engine.SurveyStageGuard(chamber, survey.Stage, next, now); err != nil {
		return model.Survey{}, fmt.Errorf("%w: %v", ErrSeasonBlocked, err)
	}
	findings := strings.TrimSpace(input.Findings)
	switch {
	case findings == "":
		findings = survey.Findings
	case survey.Findings == "":
		survey.Findings = findings
	default:
		survey.Findings = survey.Findings + "; " + findings
	}
	if next == model.SurveyReview && !survey.CompletionReady() {
		return model.Survey{}, fmt.Errorf("%w: findings and surface condition required before review", ErrInvalid)
	}
	if err := s.store.TransitionSurvey(survey.ID, survey.Stage, next, survey.Findings, now); err != nil {
		return model.Survey{}, err
	}
	s.bump("survey_transitions")
	survey.Stage = next
	survey.UpdatedAt = now
	if next == model.SurveyClosed {
		survey.CompletedAt = now
	}
	return survey, nil
}

// SurveyWithNotes returns the survey plus its conservation notes for review hand-off.
type SurveyDetail struct {
	Survey model.Survey             `json:"survey"`
	Notes  []model.ConservationNote `json:"notes"`
	Age    time.Duration            `json:"age_hours"`
}

func (s *Service) SurveyDetail(surveyID string) (SurveyDetail, error) {
	survey, err := s.GetSurvey(surveyID)
	if err != nil {
		return SurveyDetail{}, err
	}
	notes, err := s.store.ListNotesBySurvey(surveyID)
	if err != nil {
		return SurveyDetail{}, err
	}
	detail := SurveyDetail{Survey: survey, Notes: notes}
	if !survey.CompletedAt.IsZero() {
		detail.Age = s.now().Sub(survey.CompletedAt).Round(time.Hour)
	} else {
		detail.Age = s.now().Sub(survey.StartedAt).Round(time.Hour)
	}
	return detail, nil
}
