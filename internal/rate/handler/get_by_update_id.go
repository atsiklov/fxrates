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
	UpdateID  string    `json:"update_id"`
	Base      string    `json:"base"`
	Quote     string    `json:"quote"`
	Value     float64   `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

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
