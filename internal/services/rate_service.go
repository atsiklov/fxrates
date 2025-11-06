package services

import (
	"fxrates/internal/domain"
	"fxrates/internal/storage"
)

type RateService struct {
	rateRepo storage.RateRepository
}

func (s *RateService) GetRateByRefreshID() (*domain.Rate, error) {
	return s.rateRepo.GetRateByRefreshID()
}

func (s *RateService) UpdateRate() error {
	return s.rateRepo.UpdateRate()
}
