package rate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCurrencyValidator_ValidatePair_Errors(t *testing.T) {
	validator := NewValidator(map[string]struct{}{"USD": {}, "EUR": {}})

	require.Equal(t, ErrBaseRequired, validator.ValidatePair("", "EUR"))
	require.Equal(t, ErrQuoteRequired, validator.ValidatePair("USD", ""))
	require.Equal(t, ErrSameCodes, validator.ValidatePair("USD", "USD"))
	require.Equal(t, ErrBaseUnsupported, validator.ValidatePair("ABC", "EUR"))
	require.Equal(t, ErrQuoteUnsupported, validator.ValidatePair("USD", "ZZZ"))
}

func TestCurrencyValidator_ValidatePair_Success(t *testing.T) {
	validator := NewValidator(map[string]struct{}{"USD": {}, "EUR": {}})
	require.NoError(t, validator.ValidatePair("USD", "EUR"))
}

func TestNewValidator_ClonesMap(t *testing.T) {
	sourceCurrencies := map[string]struct{}{"USD": {}, "EUR": {}}
	validator := NewValidator(sourceCurrencies)

	// mutate source after creation
	delete(sourceCurrencies, "USD")

	// validator should still allow USD (clone must not be affected)
	require.NoError(t, validator.ValidatePair("USD", "EUR"))
}
