package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"fxrates/internal/services"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type RateHandler struct {
	RateService *services.RateService
}

type UpdateRateRequest struct {
	Code string `json:"code"`
}

type UpdateRateResponse struct {
	UpdateID string `json:"update_id"`
}

// UpdateRate todo: ctx, error handling, validation
func (h *RateHandler) UpdateRate(w http.ResponseWriter, r *http.Request) {
	var req UpdateRateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(422)
		_, _ = w.Write([]byte(fmt.Sprintf("invalid request param %s", err)))
		return
	}
	code := strings.ToUpper(strings.TrimSpace(req.Code))

	updateID, err := h.RateService.UpdateRateByCode(r.Context(), code)
	if err != nil {
		w.WriteHeader(422)
		_, _ = w.Write([]byte(fmt.Sprintf("error while updating rate %s", err)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(UpdateRateResponse{
		UpdateID: updateID.String(),
	})
}

type GetRateInfoResponse struct {
	UpdateID  string     `json:"update_id"`
	Code      string     `json:"code"`
	Price     float64    `json:"price"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// GetRateInfo todo: ctx, error handling, validation
func (h *RateHandler) GetRateInfo(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	updateID, err := uuid.Parse(rawID)
	if err != nil {
		w.WriteHeader(422)
		_, _ = w.Write([]byte(fmt.Sprintf("error while updating rate %s", err)))
		return
	}

	ctx := context.Background()
	rateInfo, err := h.RateService.GetRateInfoByUpdateID(ctx, updateID)
	if err != nil {
		w.WriteHeader(422)
		_, _ = w.Write([]byte(fmt.Sprintf("error while updating rate %s", err)))
		return
	}

	res := GetRateInfoResponse{
		UpdateID:  updateID.String(),
		Code:      rateInfo.Code,
		Price:     rateInfo.NewPrice,
		UpdatedAt: rateInfo.UpdatedAt,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

type GetRateResponse struct {
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	Price     float64   `json:"price"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetRate todo: ctx, error handling, validation
func (h *RateHandler) GetRate(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "code")))
	// check if valid

	ctx := r.Context()
	rate, err := h.RateService.GetRateByCode(ctx, code)
	if err != nil {
		w.WriteHeader(422)
		_, _ = w.Write([]byte(fmt.Sprintf("error while updating rate %s", err)))
		return
	}

	res := GetRateResponse{
		Name:      rate.Name,
		Code:      rate.Code,
		Price:     rate.Price,
		UpdatedAt: rate.UpdatedAt,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}
