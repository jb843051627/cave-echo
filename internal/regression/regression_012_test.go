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

func TestBug12_BatchRejectsForeignSiteSensor(t *testing.T) {
	svc := newSvc(t, metrics.New())
	siteA, _, _ := seedCave(t, svc, 0, 0)
	siteB, err := svc.CreateSite(model.CreateSiteInput{
		Code: "CAVE2", Name: "Far Cave", Latitude: -5.5, Longitude: 12.0,
		ProtectionGrade: model.ProtectionGradeA,
	})
	if err != nil {
		t.Fatal(err)
	}
	chamberB, err := svc.CreateChamber(siteB.ID, model.CreateChamberInput{Name: "Deep Hall"})
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := svc.RegisterSensor(model.CreateSensorInput{
		ChamberID: chamberB.ID, Name: "thermo-far", Type: model.SensorHumidity,
		MinValue: 0, MaxValue: 100, SampleIntervalSec: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = pushBatch(t, svc, siteA, pt(foreign.ID, baseTime.Add(-time.Minute), 40))
	if !errors.Is(err, service.ErrInvalid) {
		t.Fatalf("cross-site sensor batch err=%v, want ErrInvalid", err)
	}
	readings, err := svc.ListReadings(store.ReadingFilter{SiteID: siteA})
	if err != nil {
		t.Fatal(err)
	}
	if len(readings) != 0 {
		t.Fatalf("foreign sensor reading stored under site A: %d rows", len(readings))
	}
}
