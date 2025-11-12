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

type RatePair struct {
	Base  string
	Quote string
}

func (p RatePair) Reversed() RatePair {
	return RatePair{
		Base:  p.Quote,
		Quote: p.Base,
	}
}
