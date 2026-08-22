package model

import "time"

type Alert struct {
	ID             string        `json:"id"`
	SiteID         string        `json:"site_id"`
	ChamberID      string        `json:"chamber_id"`
	SensorID       string        `json:"sensor_id"`
	Kind           AlertKind     `json:"kind"`
	Severity       AlertSeverity `json:"severity"`
	Status         AlertStatus   `json:"status"`
	DedupKey       string        `json:"dedup_key"`
	Message        string        `json:"message"`
	FirstSeenAt    time.Time     `json:"first_seen_at"`
	LastSeenAt     time.Time     `json:"last_seen_at"`
	AcknowledgedAt time.Time     `json:"acknowledged_at"`
	ClosedAt       time.Time     `json:"closed_at"`
	Occurrences    int           `json:"occurrences"`
}

func (a Alert) CanAcknowledge() bool {
	return a.Status == AlertOpen
}

func (a Alert) CanClose() bool {
	if a.Status == AlertClosed {
		return false
	}
	return a.Severity != SeverityCritical || a.Status == AlertAcknowledged
}

func (a Alert) IsActive() bool {
	return a.Status != AlertClosed
}
