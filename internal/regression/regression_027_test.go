package regression

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/jb843051627/cave-echo/internal/cache"
	"github.com/jb843051627/cave-echo/internal/clock"
	"github.com/jb843051627/cave-echo/internal/engine"
	"github.com/jb843051627/cave-echo/internal/metrics"
	"github.com/jb843051627/cave-echo/internal/model"
	"github.com/jb843051627/cave-echo/internal/service"
	"github.com/jb843051627/cave-echo/internal/store"
)

var baseTime = time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)

func TestBug27_NoteRejectsForeignSiteSurvey(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := service.New(service.Dependencies{
		Store: st, Engine: engine.New(), Cache: cache.New(),
		Clock: clock.Fixed{Value: baseTime}, Metrics: metrics.New(),
	})
	siteA, err := svc.CreateSite(model.CreateSiteInput{
		Code: "SITE-A", Name: "Alpha Cave", Latitude: 1.5, Longitude: 2.5,
		ProtectionGrade: model.ProtectionGradeB,
	})
	if err != nil {
		t.Fatal(err)
	}
	chamberA, err := svc.CreateChamber(siteA.ID, model.CreateChamberInput{Name: "Gallery A"})
	if err != nil {
		t.Fatal(err)
	}
	surveyA, err := svc.CreateSurvey(model.CreateSurveyInput{
		SiteID: siteA.ID, ChamberID: chamberA.ID, Transect: "TA",
	})
	if err != nil {
		t.Fatal(err)
	}
	siteB, err := svc.CreateSite(model.CreateSiteInput{
		Code: "SITE-B", Name: "Beta Cave", Latitude: 3.5, Longitude: 4.5,
		ProtectionGrade: model.ProtectionGradeC,
	})
	if err != nil {
		t.Fatal(err)
	}
	chamberB, err := svc.CreateChamber(siteB.ID, model.CreateChamberInput{Name: "Gallery B"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.AddNote(model.CreateNoteInput{
		SiteID:   siteB.ID,
		ChamberID: chamberB.ID,
		SurveyID: surveyA.ID,
		Category: model.NoteInspection,
		Note:     "cross-site attachment attempt",
	})
	if !errors.Is(err, service.ErrInvalid) {
		t.Fatalf("note referencing survey of another site err=%v, want ErrInvalid", err)
	}
}
