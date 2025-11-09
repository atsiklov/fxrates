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
	GetByCodes(ctx context.Context, base string, quote string) (*domain.AppliedRate, error)
}

type RateUpdatesRepository interface {
	ScheduleNewOrGetExisting(ctx context.Context, base string, quote string) (uuid.UUID, error)
	GetByUpdateID(ctx context.Context, updateID uuid.UUID) (*domain.AppliedRate, error)
	GetPending(ctx context.Context) ([]domain.PendingRate, error)
	SaveApplied(ctx context.Context, rates []domain.AppliedRate) error
}
