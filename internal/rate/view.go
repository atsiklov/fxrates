package rate

import (
	"fxrates/internal/domain"
	"time"
)

type View struct {
	Base      string
	Quote     string
	Status    domain.RateUpdateStatus
	Value     *float64
	UpdatedAt *time.Time
}
