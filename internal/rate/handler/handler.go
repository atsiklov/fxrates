package handler

import (
	"encoding/json"
	"fxrates/internal/rate"
	"net/http"
)

type Handler struct {
	validator *rate.CurrencyValidator
	service   *rate.Service
}

func NewRateHandler(RateService *rate.Service, CurrencyValidator *rate.CurrencyValidator) *Handler {
	return &Handler{validator: CurrencyValidator, service: RateService}
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, statusCode int, errorMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error: errorMsg,
	})
}
