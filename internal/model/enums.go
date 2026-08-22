package model

type ProtectionGrade string

const (
	ProtectionGradeA ProtectionGrade = "A"
	ProtectionGradeB ProtectionGrade = "B"
	ProtectionGradeC ProtectionGrade = "C"
)

func (g ProtectionGrade) Valid() bool {
	return g == ProtectionGradeA || g == ProtectionGradeB || g == ProtectionGradeC
}

type SiteStatus string

const (
	SiteOpen       SiteStatus = "open"
	SiteRestricted SiteStatus = "restricted"
	SiteClosed     SiteStatus = "closed"
)

func (s SiteStatus) Valid() bool {
	return s == SiteOpen || s == SiteRestricted || s == SiteClosed
}

type SensorType string

const (
	SensorTemperature SensorType = "temperature"
	SensorHumidity    SensorType = "humidity"
	SensorCO2         SensorType = "co2"
	SensorPressure    SensorType = "pressure"
	SensorAcoustic    SensorType = "acoustic"
	SensorAirflow     SensorType = "airflow"
	SensorLight       SensorType = "light"
)

func (s SensorType) Valid() bool {
	switch s {
	case SensorTemperature, SensorHumidity, SensorCO2, SensorPressure, SensorAcoustic, SensorAirflow, SensorLight:
		return true
	default:
		return false
	}
}

type ReadingQuality string

const (
	QualityGood     ReadingQuality = "good"
	QualitySuspect  ReadingQuality = "suspect"
	QualityRejected ReadingQuality = "rejected"
)

func (q ReadingQuality) Valid() bool {
	return q == QualityGood || q == QualitySuspect || q == QualityRejected
}

type DripColor string

const (
	DripClear  DripColor = "clear"
	DripAmber  DripColor = "amber"
	DripCloudy DripColor = "cloudy"
)

func (c DripColor) Valid() bool {
	return c == DripClear || c == DripAmber || c == DripCloudy
}

type GasMethod string

const (
	GasSpot      GasMethod = "spot"
	GasPump      GasMethod = "pump"
	GasDiffusion GasMethod = "diffusion"
	GasLogger    GasMethod = "logger"
)

func (m GasMethod) Valid() bool {
	return m == GasSpot || m == GasPump || m == GasDiffusion || m == GasLogger
}

type SurveyStage string

const (
	SurveyPlanned   SurveyStage = "planned"
	SurveyFieldwork SurveyStage = "fieldwork"
	SurveyReview    SurveyStage = "review"
	SurveyClosed    SurveyStage = "closed"
)

func (s SurveyStage) Valid() bool {
	return s == SurveyPlanned || s == SurveyFieldwork || s == SurveyReview || s == SurveyClosed
}

func (s SurveyStage) CanMoveTo(next SurveyStage) bool {
	if !next.Valid() || s == SurveyClosed {
		return false
	}
	switch s {
	case SurveyPlanned:
		return next == SurveyFieldwork
	case SurveyFieldwork:
		return next == SurveyReview
	case SurveyReview:
		return next == SurveyClosed || next == SurveyFieldwork
	default:
		return false
	}
}

type StabilityLevel string

const (
	StabilityStable     StabilityLevel = "stable"
	StabilityWatch      StabilityLevel = "watch"
	StabilityRestricted StabilityLevel = "restricted"
	StabilityClosed     StabilityLevel = "closed"
)

func (l StabilityLevel) Valid() bool {
	return l == StabilityStable || l == StabilityWatch || l == StabilityRestricted || l == StabilityClosed
}

type AlertKind string

const (
	AlertMicroclimate AlertKind = "microclimate"
	AlertGas          AlertKind = "gas"
	AlertDrip         AlertKind = "drip"
	AlertSensor       AlertKind = "sensor_offline"
	AlertCompleteness AlertKind = "completeness"
	AlertProtection   AlertKind = "protection"
)

func (k AlertKind) Valid() bool {
	switch k {
	case AlertMicroclimate, AlertGas, AlertDrip, AlertSensor, AlertCompleteness, AlertProtection:
		return true
	default:
		return false
	}
}

type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

func (s AlertSeverity) Valid() bool {
	return s == SeverityInfo || s == SeverityWarning || s == SeverityCritical
}

type AlertStatus string

const (
	AlertOpen         AlertStatus = "open"
	AlertAcknowledged AlertStatus = "acknowledged"
	AlertClosed       AlertStatus = "closed"
)

func (s AlertStatus) Valid() bool {
	return s == AlertOpen || s == AlertAcknowledged || s == AlertClosed
}

type NoteCategory string

const (
	NoteInspection NoteCategory = "inspection"
	NoteTreatment  NoteCategory = "treatment"
	NoteAccess     NoteCategory = "access"
	NoteWildlife   NoteCategory = "wildlife"
	NoteIncident   NoteCategory = "incident"
)

func (c NoteCategory) Valid() bool {
	return c == NoteInspection || c == NoteTreatment || c == NoteAccess || c == NoteWildlife || c == NoteIncident
}
