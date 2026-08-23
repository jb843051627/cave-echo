package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jb843051627/cave-echo/internal/service"
)

const maxBodyBytes = 4 << 20

// decodeJSON parses a JSON request body into out with size limits.
func decodeJSON(w http.ResponseWriter, r *http.Request, out any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(out); err != nil {
		return service.ErrInvalid
	}
	return nil
}

func pathValue(r *http.Request, name string) string {
	return strings.TrimSpace(r.PathValue(name))
}

func queryValue(r *http.Request, name string) string {
	return strings.TrimSpace(r.URL.Query().Get(name))
}

func queryTime(r *http.Request, name string) (time.Time, error) {
	raw := queryValue(r, name)
	if raw == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, errors.New("invalid timestamp, expected RFC3339")
	}
	return parsed.UTC(), nil
}

func queryInt(r *http.Request, name string, fallback int) int {
	raw := queryValue(r, name)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func queryWindow(r *http.Request) time.Duration {
	raw := queryValue(r, "window")
	if raw == "" {
		return 24 * time.Hour
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 || parsed > 365*24*time.Hour {
		return 24 * time.Hour
	}
	return parsed
}
