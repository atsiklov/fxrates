package adapters

import (
	"context"
	"fxrates/internal/domain"

	"github.com/google/uuid"
)

type RateClient interface {
	GetExchangeRates(ctx context.Context, code string) (map[string]float64, error)
}

type RateRepository interface {
	GetByCodes(ctx context.Context, base string, quote string) (domain.Rate, error)
	GetByUpdateID(ctx context.Context, updateID uuid.UUID) (domain.Rate, domain.RateUpdateStatus, error)
}

type RateUpdateRepository interface {
	ScheduleNewOrGetExisting(ctx context.Context, base string, quote string) (uuid.UUID, error)
	GetPending(ctx context.Context) ([]domain.PendingRateUpdate, error)
	ApplyUpdates(ctx context.Context, rates []domain.AppliedRateUpdate) error
}
