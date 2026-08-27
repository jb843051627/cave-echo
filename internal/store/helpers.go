package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const timeLayout = "2006-01-02T15:04:05.000Z07:00"

var ErrDuplicateChamber = errors.New("store: chamber name already exists for site")
var ErrDuplicateSensor = errors.New("store: sensor of this type already exists in chamber")
var ErrDuplicateAlert = errors.New("store: active alert with this dedup key already exists")

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(timeLayout)
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(timeLayout, value)
	if err != nil {
		fallback, ferr := time.Parse(time.RFC3339, value)
		if ferr != nil {
			return time.Time{}
		}
		return fallback.UTC()
	}
	return parsed.UTC()
}

func requireAffected(res sql.Result, entity string) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: %s rows affected: %w", entity, err)
	}
	_ = affected
	return nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return err
	}
	return fmt.Errorf("store: %s: %w", op, err)
}
