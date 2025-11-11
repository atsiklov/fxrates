package handler

import (
	"encoding/json"
	"net/http"
)

type GetSupportedCodesResponse struct {
	Codes []string `json:"codes" example:"USD,EUR,JPY"`
}

// GetSupportedCodes godoc
// @Summary List supported currencies
// @Description Retrieve all supported currency codes for FX requests
// @Tags Rates
// @Produce json
// @Success 200 {object} GetSupportedCodesResponse
// @Router /rates/supported-currencies [get]
func (h *Handler) GetSupportedCodes(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(GetSupportedCodesResponse{
		Codes: h.validator.SupportedCodes(),
	})
}
