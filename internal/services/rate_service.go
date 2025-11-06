package services

import (
	"context"
	"fxrates/internal/domain"
	"fxrates/internal/storage"

	"github.com/google/uuid"
)

type RateService struct {
	rateRepo storage.RateRepository
}

func (s *RateService) GetRateInfoByUpdateID(ctx context.Context, updateID uuid.UUID) (*storage.RateInfo, error) {
	return s.rateRepo.GetRateInfoByUpdateID(ctx, updateID)
}

func (s *RateService) UpdateRateByCode(ctx context.Context, code string) (uuid.UUID, error) {
	return s.rateRepo.UpdateRateByCode(ctx, code)
}

func (s *RateService) GetRateByCode(ctx context.Context, code string) (*domain.Rate, error) {
	return s.rateRepo.GetRateByCode(ctx, code)
}

func NewRateService(rateRepo storage.RateRepository) *RateService {
	return &RateService{
		rateRepo: rateRepo,
	}
}
