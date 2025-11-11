package rate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCurrencyValidator_ValidatePair_Errors(t *testing.T) {
	validator := NewValidator(map[string]struct{}{"USD": {}, "EUR": {}})

	require.Equal(t, ErrBaseRequired, validator.ValidateCodes("", "EUR"))
	require.Equal(t, ErrQuoteRequired, validator.ValidateCodes("USD", ""))
	require.Equal(t, ErrSameCodes, validator.ValidateCodes("USD", "USD"))
	require.Equal(t, ErrBaseUnsupported, validator.ValidateCodes("ABC", "EUR"))
	require.Equal(t, ErrQuoteUnsupported, validator.ValidateCodes("USD", "ZZZ"))
}

func TestCurrencyValidator_ValidatePair_Success(t *testing.T) {
	validator := NewValidator(map[string]struct{}{"USD": {}, "EUR": {}})
	require.NoError(t, validator.ValidateCodes("USD", "EUR"))
}

func TestNewValidator_ClonesMap(t *testing.T) {
	sourceCurrencies := map[string]struct{}{"USD": {}, "EUR": {}}
	validator := NewValidator(sourceCurrencies)

	// mutate source after creation
	delete(sourceCurrencies, "USD")

	// validator should still allow USD (clone must not be affected)
	require.NoError(t, validator.ValidateCodes("USD", "EUR"))
}

func TestCurrencyValidator_SupportedCodes(t *testing.T) {
	validator := NewValidator(map[string]struct{}{"USD": {}, "EUR": {}, "JPY": {}})

	got := validator.SupportedCodes()

	require.Len(t, got, 3)
	require.ElementsMatch(t, []string{"USD", "EUR", "JPY"}, got)

	// ensure caller modifications do not affect validator internal state
	got[0] = "XXX"
	require.ElementsMatch(t, []string{"USD", "EUR", "JPY"}, validator.SupportedCodes())
}
