package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jb843051627/cave-echo/internal/api"
	"github.com/jb843051627/cave-echo/internal/cache"
	"github.com/jb843051627/cave-echo/internal/clock"
	"github.com/jb843051627/cave-echo/internal/config"
	"github.com/jb843051627/cave-echo/internal/engine"
	"github.com/jb843051627/cave-echo/internal/ingest"
	"github.com/jb843051627/cave-echo/internal/metrics"
	"github.com/jb843051627/cave-echo/internal/service"
	"github.com/jb843051627/cave-echo/internal/store"
	"github.com/jb843051627/cave-echo/internal/validation"
)

func main() {
	cfg := config.FromEnv()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	repository, err := store.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer repository.Close()

	registry := metrics.New()
	snapshot := cache.New()
	queue := ingest.New(cfg.IngestQueueSize)
	defer queue.Close()

	app := service.New(service.Dependencies{
		Store:       repository,
		Engine:      engine.New(),
		Cache:       snapshot,
		Clock:       clock.System{},
		Metrics:     registry,
		Queue:       queue,
		DedupWindow: cfg.DedupWindow,
		Limits: validation.Limits{
			FutureSkew:       cfg.FutureSkew,
			RetentionWindow:  cfg.RetentionWindow,
			MaxBatchReadings: 5000,
		},
	})

	server := api.New(api.Dependencies{
		Service: app,
		Metrics: registry,
		Static:  cfg.StaticDir,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Background watchdog: offline sensor detection and stale alert expiry.
	go runWatchdog(ctx, app, cfg.EvaluationPeriod, cfg.HeartbeatMaxAge())

	httpServer := &http.Server{
		Addr:              cfg.Address,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	log.Printf("cave-echo listening on %s (db=%s)", cfg.Address, cfg.DatabasePath)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// runWatchdog periodically scans for silent sensors and expires alerts that
// have stayed quiet beyond the dedup window.
func runWatchdog(ctx context.Context, app *service.Service, period, heartbeatMaxAge time.Duration) {
	if period <= 0 {
		period = time.Minute
	}
	if heartbeatMaxAge <= 0 {
		heartbeatMaxAge = 30 * time.Minute
	}
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			touched, err := app.EvaluateOfflineSensors(heartbeatMaxAge)
			if err != nil {
				log.Printf("watchdog: offline scan: %v", err)
				continue
			}
			expired, err := app.ExpireStaleAlerts()
			if err != nil {
				log.Printf("watchdog: alert expiry: %v", err)
				continue
			}
			if touched > 0 || expired > 0 {
				log.Printf("watchdog: %d offline sensor(s), %d expired alert(s)", touched, expired)
			}
		}
	}
}
