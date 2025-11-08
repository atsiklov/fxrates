package rate

import (
	"context"
	"fxrates/internal/adapters"
	"fxrates/internal/domain"

	"github.com/google/uuid"
)

type Service struct {
	repo adapters.RateRepository
}

func (s *Service) GetByUpdateID(ctx context.Context, updateID uuid.UUID) (*domain.AppliedRate, error) {
	return s.repo.GetByUpdateID(ctx, updateID)
}

func (s *Service) UpdateByCode(ctx context.Context, base string, quote string) (uuid.UUID, error) {
	return s.repo.UpdateByCode(ctx, base, quote)
}

func (s *Service) GetByCode(ctx context.Context, base string, quote string) (*domain.AppliedRate, error) {
	return s.repo.GetByCode(ctx, base, quote)
}

func NewService(repo adapters.RateRepository) *Service {
	return &Service{repo: repo}
}
