package rates

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type RateHandler struct {
	RateService *RateService
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

	updateID, err := h.RateService.UpdateByCode(r.Context(), "base", "quote") // todo
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

type GetRateResponse struct {
	UpdateID  string    `json:"update_id"`
	Value     float64   `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetByUpdateID todo: ctx, error handling, validation
func (h *RateHandler) GetByUpdateID(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	updateID, err := uuid.Parse(rawID)
	if err != nil {
		w.WriteHeader(422)
		_, _ = w.Write([]byte(fmt.Sprintf("error while updating rate %s", err)))
		return
	}

	ctx := context.Background()
	rate, err := h.RateService.GetByUpdateID(ctx, updateID)
	if err != nil {
		w.WriteHeader(422)
		_, _ = w.Write([]byte(fmt.Sprintf("error while updating rate %s", err)))
		return
	}

	res := GetRateResponse{
		UpdateID:  updateID.String(),
		Value:     rate.Value,
		UpdatedAt: rate.UpdatedAt,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

// GetRate todo: ctx, error handling, validation
func (h *RateHandler) GetRate(w http.ResponseWriter, r *http.Request) {
	// code := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "code")))
	// check if valid

	ctx := r.Context()
	rate, err := h.RateService.GetByCode(ctx, "base", "quote") // todo: implement
	if err != nil {
		w.WriteHeader(422)
		_, _ = w.Write([]byte(fmt.Sprintf("error while updating rate %s", err)))
		return
	}

	res := GetRateResponse{
		Value:     rate.Value,
		UpdatedAt: rate.UpdatedAt,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}
