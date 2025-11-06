package delivery

import (
	"context"
	"fxrates/internal/domain"
	"fxrates/internal/services"
	"fxrates/internal/storage"

	"github.com/google/uuid"
)

type RateHandler struct {
	service *services.RateService
}

// UpdateRate todo: ...
func (h *RateHandler) UpdateRate(code string) uuid.UUID {
	ctx := context.Background()
	updateID, err := h.service.UpdateRateByCode(ctx, code)
	if err != nil {
		panic(err)
	}
	return updateID
}

// GetRateInfo todo: ...
func (h *RateHandler) GetRateInfo(updateID uuid.UUID) *storage.RateInfo {
	ctx := context.Background()
	rateInfo, err := h.service.GetRateInfoByUpdateID(ctx, updateID)
	if err != nil {
		panic(err)
	}
	return rateInfo
}

// GetRate todo: ...
func (h *RateHandler) GetRate(code string) *domain.Rate {
	ctx := context.Background()
	rate, err := h.service.GetRateByCode(ctx, code)
	if err != nil {
		panic(err)
	}
	return rate
}
