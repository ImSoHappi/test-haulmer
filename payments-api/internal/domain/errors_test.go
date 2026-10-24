package domain_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

func TestDomainErrors_CatalogIntegrity(t *testing.T) {
	testCases := []struct {
		name       string
		err        *domain.DomainError
		wantCode   string
		wantStatus int
	}{
		{"InvalidCardFormat", domain.ErrInvalidCardFormat, "invalid_card_format", 422},
		{"InvalidCardLuhn", domain.ErrInvalidCardLuhn, "invalid_card_luhn", 422},
		{"CardExpired", domain.ErrCardExpired, "card_expired", 422},
		{"InvalidCVV", domain.ErrInvalidCVV, "invalid_cvv", 422},
		{"InvalidAmount", domain.ErrInvalidAmount, "invalid_amount", 400},
		{"AmountExceedsLimit", domain.ErrAmountExceedsLimit, "amount_exceeds_limit", 422},
		{"UnsupportedCurrency", domain.ErrUnsupportedCurrency, "unsupported_currency", 400},
		{"MissingRequiredField", domain.ErrMissingRequiredField, "missing_required_field", 400},
		{"MerchantNotFound", domain.ErrMerchantNotFound, "merchant_not_found", 404},
		{"TransactionNotFound", domain.ErrTransactionNotFound, "transaction_not_found", 404},
		{"MerchantInactive", domain.ErrMerchantInactive, "merchant_inactive", 422},
		{"IdempotencyConflict", domain.ErrIdempotencyConflict, "idempotency_conflict", 409},
		{"ActiveTransactionExists", domain.ErrActiveTransactionExists, "active_transaction_exists", 409},
		{"IntentAlreadyApproved", domain.ErrIntentAlreadyApproved, "intent_already_approved", 409},
		{"TerminalStateError", domain.ErrTerminalStateTransition, "terminal_state_error", 409},
		{"InvalidStateTransition", domain.ErrInvalidStateTransition, "invalid_state_transition", 422},
		{"AcquirerUnavailable", domain.ErrAcquirerUnavailable, "acquirer_unavailable", 503},
		{"AcquirerTimeout", domain.ErrAcquirerTimeout, "acquirer_timeout", 504},
		{"AcquirerUnknownResult", domain.ErrAcquirerUnknownResult, "acquirer_unknown_result", 202},
		{"AcquirerUnexpected", domain.ErrAcquirerUnexpected, "acquirer_unexpected_error", 502},
	}

	codesSeen := make(map[string]bool)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotNil(t, tc.err)
			assert.Equal(t, tc.wantCode, tc.err.Code)
			assert.Equal(t, tc.wantStatus, tc.err.HTTPStatus)
			assert.NotEmpty(t, tc.err.Message)
			assert.Equal(t, tc.err.Message, tc.err.Error(), "Error() must return message")

			// Ensure error code uniqueness across catalog
			assert.False(t, codesSeen[tc.err.Code], "duplicate error code in catalog: %s", tc.err.Code)
			codesSeen[tc.err.Code] = true

			// Ensure errors.Is works with pointer comparison
			assert.True(t, errors.Is(tc.err, tc.err))
		})
	}
}
