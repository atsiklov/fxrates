package domain

import "github.com/google/uuid"

type RateUpdateStatus string

const (
	StatusPending RateUpdateStatus = "pending"
	StatusApplied RateUpdateStatus = "applied"
)

type PendingRateUpdate struct {
	UpdateID uuid.UUID `json:"update_id"`
	PairID   int64     `json:"pair_id"`
	Base     string    `json:"base"`
	Quote    string    `json:"quote"`
}

type AppliedRateUpdate struct {
	UpdateID uuid.UUID `json:"update_id"`
	PairID   int64     `json:"pair_id"`
	Value    float64   `json:"value"`
}
