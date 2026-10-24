package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

func TestAmount_ValidAmounts(t *testing.T) {
	validValues := []int64{1, 100, 25990, 5000000}
	for _, val := range validValues {
		amt, err := domain.NewAmount(val)
		require.NoError(t, err)
		assert.Equal(t, val, amt.Int64())
	}
}

func TestAmount_InvalidAmounts(t *testing.T) {
	invalidValues := []int64{0, -1, -5000}
	for _, val := range invalidValues {
		_, err := domain.NewAmount(val)
		assert.ErrorIs(t, err, domain.ErrInvalidAmount)
	}
}

func TestAmount_ValidateMax(t *testing.T) {
	amt, err := domain.NewAmount(25000)
	require.NoError(t, err)

	// Within limit
	assert.NoError(t, amt.ValidateMax(50000))

	// Exceeds limit
	assert.ErrorIs(t, amt.ValidateMax(10000), domain.ErrAmountExceedsLimit)
}

func TestCurrency_Validation(t *testing.T) {
	assert.Equal(t, "CLP", domain.CurrencyCLP)

	// Valid CLP
	curr, err := domain.NewCurrency("CLP")
	require.NoError(t, err)
	assert.Equal(t, domain.CurrencyCLP, curr)

	// Invalid currencies
	_, err = domain.NewCurrency("USD")
	assert.ErrorIs(t, err, domain.ErrUnsupportedCurrency)

	_, err = domain.NewCurrency("")
	assert.ErrorIs(t, err, domain.ErrUnsupportedCurrency)
}
