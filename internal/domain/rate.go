package domain

import "time"

type Rate struct {
	ID         int64
	Name       string
	Code       string
	Price      float64
	CreateDate time.Time
	UpdateDate time.Time
}
