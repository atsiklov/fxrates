package domain

import "time"

type AppliedRate struct {
	PairID    int64
	Base      string
	Quote     string
	Value     float64
	UpdatedAt time.Time
}

type PendingRate struct {
	PairID int64
	Base   string
	Quote  string
}
