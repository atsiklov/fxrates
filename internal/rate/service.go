package rate

import (
	"context"
	"fmt"
	"fxrates/internal/adapters"
	"fxrates/internal/domain"

	"github.com/google/uuid"
)

type Service struct {
	rateUpdatesRepo adapters.RateUpdateRepository
	rateRepo        adapters.RateRepository
	cache           adapters.RateUpdateCache
}

// ScheduleUpdate checks if pair presents in cache first, otherwise goes to DB
func (s *Service) ScheduleUpdate(ctx context.Context, base string, quote string) (uuid.UUID, error) {
	pair := domain.RatePair{Base: base, Quote: quote}
	if cachedID, ok := s.cache.Get(pair); ok {
		return cachedID, nil
	}

	updateID, err := s.rateUpdatesRepo.ScheduleNewOrGetExisting(ctx, base, quote)
	if err != nil {
		return uuid.Nil, err
	}

	s.cache.Set(pair, updateID)
	return updateID, nil
}

// GetByUpdateID defines View structure depending on update status and returns it
func (s *Service) GetByUpdateID(ctx context.Context, updateID uuid.UUID) (View, error) {
	rate, status, err := s.rateRepo.GetByUpdateID(ctx, updateID)
	if err != nil {
		return View{}, err
	}
	switch status {
	case domain.StatusApplied:
		return View{
			Base:      rate.Base,
			Quote:     rate.Quote,
			Status:    status,
			Value:     &rate.Value,     // never nil (DB constraint)
			UpdatedAt: &rate.UpdatedAt, // never nil (DB constraint)
		}, nil
	case domain.StatusPending:
		return View{
			Base:   rate.Base,
			Quote:  rate.Quote,
			Status: status,
		}, nil
	default:
		return View{}, fmt.Errorf("unknown rate update status: %q", status)
	}
}

func (s *Service) GetByCodes(ctx context.Context, base string, quote string) (View, error) {
	rate, err := s.rateRepo.GetByCodes(ctx, base, quote)
	if err != nil {
		return View{}, err
	}
	return View{Base: rate.Base, Quote: rate.Quote, Value: &rate.Value, UpdatedAt: &rate.UpdatedAt}, nil
}

func NewService(rateUpdatesRepo adapters.RateUpdateRepository, rateRepo adapters.RateRepository, cache adapters.RateUpdateCache) *Service {
	return &Service{
		rateUpdatesRepo: rateUpdatesRepo,
		rateRepo:        rateRepo,
		cache:           cache,
	}
}
