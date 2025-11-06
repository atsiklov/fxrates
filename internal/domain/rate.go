package domain

import "time"

type Rate struct {
	ID        int64
	Name      string
	Code      string
	Price     float64
	CreatedAt time.Time
	UpdatedAt time.Time
}
