package storage

import (
	"fxrates/internal/domain"
)

type RateRepository interface {
	UpdateRate() error
	GetRateByRefreshID() (*domain.Rate, error)
	GetRateByCode(code string) (*domain.Rate, error)
}
