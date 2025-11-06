package delivery

import (
	"fxrates/internal/domain"
	"fxrates/internal/services"
)

type RateHandler struct {
	service *services.RateService
}

// UpdateRate todo: ...
func (h *RateHandler) UpdateRate() {
	err := h.service.UpdateRate()
	if err != nil {
		panic(err)
	}
}

// GetRate todo: ...
func (h *RateHandler) GetRate() *domain.Rate {
	rate, err := h.service.GetRateByRefreshID()
	if err != nil {
		panic(err)
	}
	return rate
}
