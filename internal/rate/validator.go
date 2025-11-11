package rate

import (
	"errors"
	"maps"
	"slices"
)

var (
	ErrBaseRequired     = errors.New("base currency is required")
	ErrQuoteRequired    = errors.New("quote currency is required")
	ErrSameCodes        = errors.New("base and quote must be different")
	ErrBaseUnsupported  = errors.New("base currency not supported")
	ErrQuoteUnsupported = errors.New("quote currency not supported")
)

type CurrencyValidator struct {
	supportedCodesSet map[string]struct{} // read only copy
	supportedCodesLst []string            // read only copy
}

func (v *CurrencyValidator) ValidateCodes(base, quote string) error {
	if base == "" {
		return ErrBaseRequired
	}
	if quote == "" {
		return ErrQuoteRequired
	}
	if base == quote {
		return ErrSameCodes
	}
	if _, ok := v.supportedCodesSet[base]; !ok {
		return ErrBaseUnsupported
	}
	if _, ok := v.supportedCodesSet[quote]; !ok {
		return ErrQuoteUnsupported
	}
	return nil
}

func (v *CurrencyValidator) SupportedCodes() []string {
	return slices.Clone(v.supportedCodesLst)
}

func NewValidator(supportedCurrencies map[string]struct{}) *CurrencyValidator {
	codesSet := maps.Clone(supportedCurrencies)
	codesLst := slices.Collect(maps.Keys(codesSet))
	slices.Sort(codesLst)

	return &CurrencyValidator{
		supportedCodesSet: codesSet,
		supportedCodesLst: codesLst,
	}
}
