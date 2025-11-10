package domain

import (
	"time"

	"github.com/google/uuid"
)

type Rate struct {
	PairID    int64
	Base      string
	Quote     string
	Value     float64
	UpdatedAt time.Time
}

type RateUpdateStatus string

const (
	StatusPending RateUpdateStatus = "pending"
	StatusApplied RateUpdateStatus = "applied"
)

type PendingRateUpdate struct {
	UpdateID uuid.UUID
	PairID   int64
	Base     string
	Quote    string
}

type AppliedRateUpdate struct {
	UpdateID uuid.UUID
	PairID   int64
	Value    float64
}
