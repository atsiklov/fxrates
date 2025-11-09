package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

type ScheduleUpdateRequest struct {
	Base  string `json:"base"`
	Quote string `json:"quote"`
}

type ScheduleUpdateResponse struct {
	UpdateID string `json:"update_id"`
}

func (h *Handler) ScheduleUpdate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 256)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req ScheduleUpdateRequest
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	base := strings.ToUpper(strings.TrimSpace(req.Base))
	quote := strings.ToUpper(strings.TrimSpace(req.Quote))

	if err := h.validator.ValidateCurrencyPair(base, quote); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	updateID, err := h.service.ScheduleUpdate(r.Context(), base, quote)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{"handler": "ScheduleUpdate", "base": base, "quote": quote}).Error("update wasn't scheduled")
		writeError(w, http.StatusInternalServerError, "failed to schedule rate update")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(ScheduleUpdateResponse{
		UpdateID: updateID.String(),
	})
}
