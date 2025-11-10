package domain

import (
	"time"
)

type Rate struct {
	PairID    int64
	Base      string
	Quote     string
	Value     float64
	UpdatedAt time.Time
}
