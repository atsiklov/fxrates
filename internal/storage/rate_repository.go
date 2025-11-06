package storage

import (
	"context"
	"fxrates/internal/domain"

	"github.com/google/uuid"
)

type RateRepository interface {
	UpdateRateByCode(ctx context.Context, code string) (uuid.UUID, error)
	GetRateInfoByUpdateID(ctx context.Context, updateID uuid.UUID) (*RateInfo, error)
	GetRateByCode(ctx context.Context, code string) (*domain.Rate, error)
}
