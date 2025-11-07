package rates

import (
	"context"
	"fxrates/internal/domain"

	"github.com/google/uuid"
)

type RateService struct {
	Repo Repository
}

func (s *RateService) GetByUpdateID(ctx context.Context, updateID uuid.UUID) (*domain.AppliedRate, error) {
	return s.Repo.GetByUpdateID(ctx, updateID)
}

func (s *RateService) UpdateByCode(ctx context.Context, base string, quote string) (uuid.UUID, error) {
	return s.Repo.UpdateByCode(ctx, base, quote)
}

func (s *RateService) GetByCode(ctx context.Context, base string, quote string) (*domain.AppliedRate, error) {
	return s.Repo.GetByCode(ctx, base, quote)
}
