package model

import "time"

type Reading struct {
	ID         string         `json:"id"`
	SensorID   string         `json:"sensor_id"`
	SensorType SensorType     `json:"sensor_type"`
	SiteID     string         `json:"site_id"`
	ChamberID  string         `json:"chamber_id"`
	ObservedAt time.Time      `json:"observed_at"`
	RawValue   float64        `json:"raw_value"`
	Value      float64        `json:"value"`
	Unit       string         `json:"unit"`
	Quality    ReadingQuality `json:"quality"`
	BatchID    string         `json:"batch_id"`
	Checksum   uint32         `json:"checksum"`
	ReceivedAt time.Time      `json:"received_at"`
}

type ReadingWindow struct {
	Items        []Reading `json:"items"`
	From         time.Time `json:"from"`
	To           time.Time `json:"to"`
	Expected     int       `json:"expected"`
	Accepted     int       `json:"accepted"`
	Completeness float64   `json:"completeness"`
}

func (w ReadingWindow) Complete() bool {
	return w.Expected > 0 && w.Completeness >= 0.9
}
