package rates

import (
	"context"
	"fxrates/internal/domain"

	"github.com/google/uuid"
)

type Repository interface {
	UpdateByCode(ctx context.Context, base string, quote string) (uuid.UUID, error)
	GetByUpdateID(ctx context.Context, updateID uuid.UUID) (*domain.AppliedRate, error)
	GetByCode(ctx context.Context, base string, quote string) (*domain.AppliedRate, error)
}

type UpdatesRepository interface {
	GetPending(ctx context.Context) ([]domain.PendingRate, error)
	SaveApplied(ctx context.Context, rates []domain.AppliedRate) error
}
