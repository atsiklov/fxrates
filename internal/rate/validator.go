package rate

import (
	"errors"
	"maps"
)

type CurrencyValidator struct {
	supportedCurrencies map[string]struct{} // READ ONLY COPY
}

var (
	ErrBaseRequired     = errors.New("base currency is required")
	ErrQuoteRequired    = errors.New("quote currency is required")
	ErrSameCodes        = errors.New("base and quote must be different")
	ErrBaseUnsupported  = errors.New("base currency not supported")
	ErrQuoteUnsupported = errors.New("quote currency not supported")
)

func (v *CurrencyValidator) ValidatePair(base, quote string) error {
	if base == "" {
		return ErrBaseRequired
	}
	if quote == "" {
		return ErrQuoteRequired
	}
	if base == quote {
		return ErrSameCodes
	}
	if _, ok := v.supportedCurrencies[base]; !ok {
		return ErrBaseUnsupported
	}
	if _, ok := v.supportedCurrencies[quote]; !ok {
		return ErrQuoteUnsupported
	}
	return nil
}

func NewValidator(supportedCurrencies map[string]struct{}) *CurrencyValidator {
	return &CurrencyValidator{supportedCurrencies: maps.Clone(supportedCurrencies)}
}
