package handler

import (
	"context"
	"encoding/json"
	"fxrates/internal/rate"
	"net/http"

	"github.com/google/uuid"
)

type CurrencyValidator interface {
	ValidateCodes(base, quote string) error
	SupportedCodes() []string
}

type RateService interface {
	ScheduleUpdate(ctx context.Context, base, quote string) (uuid.UUID, error)
	GetByUpdateID(ctx context.Context, id uuid.UUID) (rate.View, error)
	GetByCodes(ctx context.Context, base, quote string) (rate.View, error)
}

type Handler struct {
	validator CurrencyValidator
	service   RateService
}

func NewRateHandler(currencyValidator CurrencyValidator, rateService RateService) *Handler {
	return &Handler{validator: currencyValidator, service: rateService}
}

type errorResponse struct {
	Error string `json:"error" example:"something bad happened"`
}

func writeError(w http.ResponseWriter, statusCode int, errorMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error: errorMsg,
	})
}
