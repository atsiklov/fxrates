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
	Base      string    `json:"base"`
	Quote     string    `json:"quote"`
	Value     float64   `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (h *Handler) GetByCodes(w http.ResponseWriter, r *http.Request) {
	base := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "base")))
	quote := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "quote")))

	if err := h.validator.ValidateCurrencyPair(base, quote); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	rate, err := h.service.GetByCodes(r.Context(), base, quote)
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
		Value:     rate.Value,
		UpdatedAt: rate.UpdatedAt,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}
