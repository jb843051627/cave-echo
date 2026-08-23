package regression

import (
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

func newSvc(t *testing.T, reg *metrics.Registry) *service.Service {
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return service.New(service.Dependencies{
		Store:   st,
		Engine:  engine.New(),
		Cache:   cache.New(),
		Clock:   clock.Fixed{Value: baseTime},
		Metrics: reg,
	})
}

func seedCave(t *testing.T, svc *service.Service, warn, crit float64) (siteID, chamberID, sensorID string) {
	site, err := svc.CreateSite(model.CreateSiteInput{
		Code: "CAVE1", Name: "Echo Cave", Latitude: 42.5, Longitude: 7.1,
		ProtectionGrade: model.ProtectionGradeB,
	})
	if err != nil {
		t.Fatal(err)
	}
	chamber, err := svc.CreateChamber(site.ID, model.CreateChamberInput{Name: "Main Gallery"})
	if err != nil {
		t.Fatal(err)
	}
	sensor, err := svc.RegisterSensor(model.CreateSensorInput{
		ChamberID:         chamber.ID,
		Name:              "thermo-1",
		Type:              model.SensorTemperature,
		Unit:              "°C",
		MinValue:          0,
		MaxValue:          100,
		WarningThreshold:  warn,
		CriticalThreshold: crit,
		SampleIntervalSec: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	return site.ID, chamber.ID, sensor.ID
}

func pushBatch(t *testing.T, svc *service.Service, siteID string, pts ...model.TelemetryPoint) (int, error) {
	t.Helper()
	batch := model.TelemetryBatch{BatchID: "batch-1", SiteID: siteID, Readings: pts}
	batch.Checksum = service.ExpectedChecksum(batch.Readings)
	return svc.IngestBatch(batch)
}

func pt(sensorID string, at time.Time, value float64) model.TelemetryPoint {
	return model.TelemetryPoint{SensorID: sensorID, ObservedAt: at, Value: value}
}

func TestBug16_StaleAlertExpiresAfterQuietWindow(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	mk := func(at time.Time) *service.Service {
		return service.New(service.Dependencies{
			Store: st, Engine: engine.New(), Cache: cache.New(),
			Clock: clock.Fixed{Value: at}, Metrics: metrics.New(),
		})
	}
	svc := mk(baseTime)
	siteID, _, sensorID := seedCave(t, svc, 50, 0)
	if _, err := pushBatch(t, svc, siteID, pt(sensorID, baseTime.Add(-time.Minute), 60)); err != nil {
		t.Fatal(err)
	}
	open, err := svc.ListAlerts(siteID, "", "", 100)
	if err != nil || len(open) != 1 {
		t.Fatalf("precondition: open alerts=%d err=%v", len(open), err)
	}
	later := mk(baseTime.Add(8 * time.Hour))
	closed, err := later.ExpireStaleAlerts()
	if err != nil {
		t.Fatal(err)
	}
	if closed < 1 {
		t.Fatalf("stale alert not expired after quiet window: closed=%d, want >= 1", closed)
	}
}
