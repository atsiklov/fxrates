package handler

import (
	"encoding/json"
	"errors"
	"fxrates/internal/domain"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type GetByCodesResponse struct {
	Base      string    `json:"base" example:"USD"`
	Quote     string    `json:"quote" example:"EUR"`
	Value     float64   `json:"value" example:"0.9231"`
	UpdatedAt time.Time `json:"updated_at" example:"2025-01-02T15:04:05Z"`
}

// GetByCodes godoc
// @Summary Get latest rate by codes
// @Description Get the latest applied FX rate by base/quote codes
// @Tags Rates
// @Produce json
// @Param base path string true "Base currency code" example(USD)
// @Param quote path string true "Quote currency code" example(EUR)
// @Success 200 {object} GetByCodesResponse
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /rates/{base}/{quote} [get]
func (h *Handler) GetByCodes(w http.ResponseWriter, r *http.Request) {
	base := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "base")))
	quote := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "quote")))

	if err := h.validator.ValidatePair(base, quote); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	view, err := h.service.GetByCodes(r.Context(), base, quote)
	if err != nil {
		if errors.Is(err, domain.ErrRateNotFound) {
			writeError(w, http.StatusNotFound, "rate not found")
			return
		}
		msg := "ups, couldn't get rate by codes this time"
		logrus.WithError(err).WithFields(logrus.Fields{"handler": "GetByCodes", "base": base, "quote": quote}).Error(msg)
		writeError(w, http.StatusInternalServerError, msg)
		return
	}

	res := GetByCodesResponse{
		Base:      base,
		Quote:     quote,
		Value:     *view.Value,
		UpdatedAt: *view.UpdatedAt,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}
