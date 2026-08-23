package model

import "time"

type ConservationNote struct {
	ID            string       `json:"id"`
	SiteID        string       `json:"site_id"`
	ChamberID     string       `json:"chamber_id"`
	SurveyID      string       `json:"survey_id"`
	Author        string       `json:"author"`
	Category      NoteCategory `json:"category"`
	Note          string       `json:"note"`
	ActionOutcome string       `json:"action_outcome"`
	CreatedAt     time.Time    `json:"created_at"`
}
