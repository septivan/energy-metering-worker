package db

import (
	"time"

	"github.com/google/uuid"
)

// MeterClient represents a meter client in the database
type MeterClient struct {
	ID                uuid.UUID
	ClientFingerprint string
	IPAddress         string
	UserAgent         *string
	FirstSeenAt       time.Time
	LastSeenAt        time.Time
	CreatedAt         time.Time
}

// MeterReading represents a meter reading in the database
type MeterReading struct {
	ID               uuid.UUID
	ClientID         uuid.UUID
	MetricName       string
	MetricValue      float64
	ReadingTimestamp time.Time
	ReceivedAt       time.Time
	ValidationStatus string
	AnomalyReason    *string
	RawPayload       []byte
}
