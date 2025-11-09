package rate

import (
	"context"
	"fxrates/internal/adapters"
	"fxrates/internal/domain"

	"github.com/google/uuid"
)

type Service struct {
	rateUpdatesRepo adapters.RateUpdatesRepository
	rateRepo        adapters.RateRepository
}

func (s *Service) ScheduleUpdate(ctx context.Context, base string, quote string) (uuid.UUID, error) {
	return s.rateUpdatesRepo.ScheduleNewOrGetExisting(ctx, base, quote)
}

func (s *Service) GetByUpdateID(ctx context.Context, updateID uuid.UUID) (*domain.AppliedRate, error) {
	return s.rateUpdatesRepo.GetByUpdateID(ctx, updateID)
}

func (s *Service) GetByCodes(ctx context.Context, base string, quote string) (*domain.AppliedRate, error) {
	return s.rateRepo.GetByCodes(ctx, base, quote)
}

func NewService(RateUpdatesRepo adapters.RateUpdatesRepository, RateRepo adapters.RateRepository) *Service {
	return &Service{rateUpdatesRepo: RateUpdatesRepo, rateRepo: RateRepo}
}
