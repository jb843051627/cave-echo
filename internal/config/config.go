package config

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultAddress        = ":8080"
	DefaultDatabasePath   = "data/cave-echo.db"
	DefaultQueueSize      = 256
	DefaultWorkers        = 4
	DefaultDedupWindow    = 6 * time.Hour
	DefaultHeartbeatGrace = 3
)

type Config struct {
	Address          string
	DatabasePath     string
	StaticDir        string
	IngestQueueSize  int
	IngestWorkers    int
	DedupWindow      time.Duration
	HeartbeatFactor  int
	RetentionWindow  time.Duration
	FutureSkew       time.Duration
	EvaluationPeriod time.Duration
}

func FromEnv() Config {
	return Config{
		Address:          envString("CAVE_ECHO_ADDR", DefaultAddress),
		DatabasePath:     envString("CAVE_ECHO_DB", DefaultDatabasePath),
		StaticDir:        envString("CAVE_ECHO_STATIC", "web/static"),
		IngestQueueSize:  envInt("CAVE_ECHO_QUEUE_SIZE", DefaultQueueSize),
		IngestWorkers:    envInt("CAVE_ECHO_WORKERS", DefaultWorkers),
		DedupWindow:      envDuration("CAVE_ECHO_DEDUP_WINDOW", DefaultDedupWindow),
		HeartbeatFactor:  envInt("CAVE_ECHO_HEARTBEAT_FACTOR", DefaultHeartbeatGrace),
		RetentionWindow:  envDuration("CAVE_ECHO_RETENTION", 30*24*time.Hour),
		FutureSkew:       envDuration("CAVE_ECHO_FUTURE_SKEW", 5*time.Minute),
		EvaluationPeriod: envDuration("CAVE_ECHO_EVAL_PERIOD", time.Minute),
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Address) == "" {
		return errors.New("config: address is empty")
	}
	if c.DatabasePath == "" {
		return errors.New("config: database path is empty")
	}
	if c.DatabasePath == ":memory:" {
		return errors.New("config: in-memory database is not allowed")
	}
	if c.IngestQueueSize <= 0 {
		return errors.New("config: ingest queue size must be positive")
	}
	if c.IngestWorkers <= 0 {
		return errors.New("config: ingest workers must be positive")
	}
	if c.DedupWindow <= 0 {
		return errors.New("config: dedup window must be positive")
	}
	if dir := filepath.Dir(c.DatabasePath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return errors.New("config: cannot create database directory: " + err.Error())
		}
	}
	return nil
}

func (c Config) HeartbeatMaxAge() time.Duration {
	factor := c.HeartbeatFactor
	if factor < 2 {
		factor = 2
	}
	return time.Duration(factor) * 10 * time.Minute
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
