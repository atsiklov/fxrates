package services

import (
	"context"
	"fxrates/internal/domain"
	"fxrates/internal/storage"

	"github.com/google/uuid"
)

type RateService struct {
	RateRepo storage.RateRepository
}

func (s *RateService) GetRateInfoByUpdateID(ctx context.Context, updateID uuid.UUID) (*storage.RateInfo, error) {
	return s.RateRepo.GetRateInfoByUpdateID(ctx, updateID)
}

func (s *RateService) UpdateRateByCode(ctx context.Context, code string) (uuid.UUID, error) {
	return s.RateRepo.UpdateRateByCode(ctx, code)
}

func (s *RateService) GetRateByCode(ctx context.Context, code string) (*domain.Rate, error) {
	return s.RateRepo.GetRateByCode(ctx, code)
}
