package regression

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jb843051627/cave-echo/internal/cache"
	"github.com/jb843051627/cave-echo/internal/clock"
	"github.com/jb843051627/cave-echo/internal/engine"
	"github.com/jb843051627/cave-echo/internal/model"
	"github.com/jb843051627/cave-echo/internal/service"
	"github.com/jb843051627/cave-echo/internal/store"
)

var baseTime = time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)

func TestBug13_NilMetricsRegistryIsTolerated(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("nil metrics registry panicked: %v", r)
		}
	}()
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := service.New(service.Dependencies{
		Store:  st,
		Engine: engine.New(),
		Cache:  cache.New(),
		Clock:  clock.Fixed{Value: baseTime},
	})
	if _, err := svc.CreateSite(model.CreateSiteInput{
		Code: "CAVE9", Name: "Silent Metrics Cave", Latitude: 1.5, Longitude: 2.5,
		ProtectionGrade: model.ProtectionGradeB,
	}); err != nil {
		t.Fatal(err)
	}
}
