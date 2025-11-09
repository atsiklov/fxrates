package handler

import (
	"encoding/json"
	"errors"
	"fxrates/internal/domain"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type GetByUpdateIDResponse struct {
	UpdateID  string    `json:"update_id" example:"77b5d9f5-0569-47e3-aee2-f659d59fbd97"`
	Base      string    `json:"base" example:"USD"`
	Quote     string    `json:"quote" example:"EUR"`
	Value     float64   `json:"value" example:"0.9231"`
	UpdatedAt time.Time `json:"updated_at" example:"2025-01-02T15:04:05Z"`
}

// GetByUpdateID godoc
// @Summary Get rate by update ID
// @Description Get the applied rate for a scheduled update ID
// @Tags Rates
// @Produce json
// @Param id path string true "Update ID"
// @Success 200 {object} GetByUpdateIDResponse
// @Failure 202 {object} errorResponse "rate update pending"
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /rates/updates/{id} [get]
func (h *Handler) GetByUpdateID(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	updateID, err := uuid.Parse(rawID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid update ID format")
		return
	}

	rate, err := h.service.GetByUpdateID(r.Context(), updateID)
	if err != nil {
		if errors.Is(err, domain.ErrRateNotFound) {
			writeError(w, http.StatusNotFound, "rate update not found")
			return
		}
		if errors.Is(err, domain.ErrRateNotApplied) {
			writeError(w, http.StatusAccepted, "rate update pending")
			return
		}
		msg := "ups, couldn't get rate by update id this time"
		logrus.WithError(err).WithFields(logrus.Fields{"handler": "GetByUpdateID", "update_id": updateID}).Error(msg)
		writeError(w, http.StatusInternalServerError, msg)
		return
	}

	res := GetByUpdateIDResponse{
		UpdateID:  updateID.String(),
		Base:      rate.Base,
		Quote:     rate.Quote,
		Value:     rate.Value,
		UpdatedAt: rate.UpdatedAt,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}
