package domain_test

import (
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

func TestCardNumber_ValidLuhn(t *testing.T) {
	validPANs := []string{
		"4242424242424242",
		"4000000000009995",
		"4000000000000119",
		"4000000000000341",
		"4000000000000101",
		"4000000000000259",
		"4532000000001239",
		"4242 4242 4242 4242", // with spaces
		"4242-4242-4242-4242", // with dashes
	}

	for _, pan := range validPANs {
		cardNum, err := domain.NewCardNumber(pan)
		require.NoError(t, err, "PAN %s should be valid", pan)

		clean := strings.ReplaceAll(strings.ReplaceAll(pan, " ", ""), "-", "")
		assert.Equal(t, clean[:6], cardNum.BIN())
		assert.Equal(t, clean[len(clean)-4:], cardNum.Last4())
		assert.Equal(t, 4, len(cardNum.Last4()))
		assert.Equal(t, 6, len(cardNum.BIN()))
	}
}

func TestCardNumber_InvalidLuhn(t *testing.T) {
	invalidLuhnPANs := []string{
		"4242424242424243",
		"4000000000009990",
		"4532000000001234",
	}

	for _, pan := range invalidLuhnPANs {
		_, err := domain.NewCardNumber(pan)
		assert.ErrorIs(t, err, domain.ErrInvalidCardLuhn, "PAN %s should fail Luhn", pan)
	}
}

func TestCardNumber_InvalidFormat(t *testing.T) {
	invalidFormats := []struct {
		pan  string
		name string
	}{
		{"", "empty"},
		{"123456789012", "too short (12 digits)"},
		{"12345678901234567890", "too long (20 digits)"},
		{"424242424242424a", "contains letters"},
		{"42424242!2424242", "contains symbols"},
	}

	for _, tc := range invalidFormats {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewCardNumber(tc.pan)
			assert.ErrorIs(t, err, domain.ErrInvalidCardFormat)
		})
	}
}

func TestCardNumber_PCI_Masking(t *testing.T) {
	cardNum, err := domain.NewCardNumber("4242424242424242")
	require.NoError(t, err)

	// BIN and Last4
	assert.Equal(t, "424242", cardNum.BIN())
	assert.Equal(t, "4242", cardNum.Last4())

	// Mask
	expectedMask := "424242******4242"
	assert.Equal(t, expectedMask, cardNum.Mask())

	// Stringer
	strOutput := fmt.Sprintf("%s", cardNum)
	assert.Equal(t, expectedMask, strOutput)
	assert.NotContains(t, strOutput, "4242424242424242")

	// slog.LogValuer
	logVal := cardNum.LogValue()
	assert.Equal(t, slog.KindString, logVal.Kind())
	assert.Equal(t, expectedMask, logVal.String())
}

func TestCVV_ValidationAndMasking(t *testing.T) {
	// Valid CVVs
	validCVVs := []string{"123", "987", "1234"}
	for _, raw := range validCVVs {
		cvv, err := domain.NewCVV(raw)
		require.NoError(t, err)
		assert.Equal(t, raw, cvv.Raw())
		assert.Equal(t, "[REDACTED]", cvv.String())

		logVal := cvv.LogValue()
		assert.Equal(t, slog.KindString, logVal.Kind())
		assert.Equal(t, "[REDACTED]", logVal.String())
	}

	// Invalid CVVs
	invalidCVVs := []string{"", "1", "12", "12345", "12a", "abc"}
	for _, raw := range invalidCVVs {
		_, err := domain.NewCVV(raw)
		assert.ErrorIs(t, err, domain.ErrInvalidCVV)
	}
}

func TestCardExpiry_ValidationAndExpiration(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

	// Valid future
	expFuture, err := domain.NewCardExpiry(12, 2028)
	require.NoError(t, err)
	assert.False(t, expFuture.IsExpired(now))

	// Normalizing 2 digits year (28 -> 2028)
	expTwoDigit, err := domain.NewCardExpiry(12, 28)
	require.NoError(t, err)
	assert.Equal(t, 2028, expTwoDigit.Year)
	assert.False(t, expTwoDigit.IsExpired(now))

	// Valid current month (expires at end of month)
	expCurrentMonth, err := domain.NewCardExpiry(10, 2026)
	require.NoError(t, err)
	assert.False(t, expCurrentMonth.IsExpired(now), "card should be valid until end of current month")

	// Expired last month
	expPastMonth, err := domain.NewCardExpiry(9, 2026)
	require.NoError(t, err)
	assert.True(t, expPastMonth.IsExpired(now))

	// Expired past year
	expPastYear, err := domain.NewCardExpiry(12, 2024)
	require.NoError(t, err)
	assert.True(t, expPastYear.IsExpired(now))

	// Invalid months
	_, err = domain.NewCardExpiry(0, 2028)
	assert.ErrorIs(t, err, domain.ErrCardExpired)

	_, err = domain.NewCardExpiry(13, 2028)
	assert.ErrorIs(t, err, domain.ErrCardExpired)
}

func TestCard_AggregationAndPCI(t *testing.T) {
	cardNum, err := domain.NewCardNumber("4242424242424242")
	require.NoError(t, err)

	cvv, err := domain.NewCVV("123")
	require.NoError(t, err)

	expiry, err := domain.NewCardExpiry(12, 2028)
	require.NoError(t, err)

	card := domain.NewCard(cardNum, expiry, cvv, "Ana Pérez")

	// Stringer check
	strCard := card.String()
	assert.Contains(t, strCard, "424242******4242")
	assert.NotContains(t, strCard, "4242424242424242")
	assert.NotContains(t, strCard, "123")

	// LogValue check
	logVal := card.LogValue()
	assert.Equal(t, slog.KindGroup, logVal.Kind())
}
