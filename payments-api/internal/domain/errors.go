package domain

import "fmt"

// DomainError represents a structured, strongly-typed domain error.
// It provides a machine-readable Code for consumers/frontends and a technical Message for developers.
type DomainError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *DomainError) Error() string {
	return e.Message
}

func (e *DomainError) String() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Validation Errors (400 / 422)
var (
	ErrInvalidCardFormat = &DomainError{
		Code:       "invalid_card_format",
		Message:    "Card number must contain only digits and be between 13 and 19 characters.",
		HTTPStatus: 422,
	}
	ErrInvalidCardLuhn = &DomainError{
		Code:       "invalid_card_luhn",
		Message:    "Card number failed Luhn checksum validation (Mod 10).",
		HTTPStatus: 422,
	}
	ErrCardExpired = &DomainError{
		Code:       "card_expired",
		Message:    "Card expiration date is in the past.",
		HTTPStatus: 422,
	}
	ErrInvalidCVV = &DomainError{
		Code:       "invalid_cvv",
		Message:    "Card verification value (CVV) must be 3 or 4 digits.",
		HTTPStatus: 422,
	}
	ErrInvalidAmount = &DomainError{
		Code:       "invalid_amount",
		Message:    "Amount must be a positive integer in CLP without decimals.",
		HTTPStatus: 400,
	}
	ErrAmountExceedsLimit = &DomainError{
		Code:       "amount_exceeds_limit",
		Message:    "Transaction amount exceeds the maximum allowable limit.",
		HTTPStatus: 422,
	}
	ErrUnsupportedCurrency = &DomainError{
		Code:       "unsupported_currency",
		Message:    "Currency is not supported; only CLP is accepted.",
		HTTPStatus: 400,
	}
	ErrMissingRequiredField = &DomainError{
		Code:       "missing_required_field",
		Message:    "A mandatory request payload field is missing or empty.",
		HTTPStatus: 400,
	}
)

// Business Rule Errors (404 / 409 / 422)
var (
	ErrMerchantNotFound = &DomainError{
		Code:       "merchant_not_found",
		Message:    "Specified merchant identifier does not exist.",
		HTTPStatus: 404,
	}
	ErrMerchantInactive = &DomainError{
		Code:       "merchant_inactive",
		Message:    "Merchant is currently inactive and cannot process payments.",
		HTTPStatus: 422,
	}
	ErrIdempotencyConflict = &DomainError{
		Code:       "idempotency_conflict",
		Message:    "Idempotency key reused with different request payload parameters.",
		HTTPStatus: 409,
	}
	ErrActiveTransactionExists = &DomainError{
		Code:       "active_transaction_exists",
		Message:    "Payment intent already has an active in-flight transaction.",
		HTTPStatus: 409,
	}
	ErrIntentAlreadyApproved = &DomainError{
		Code:       "intent_already_approved",
		Message:    "Payment intent has already been fulfilled by an approved transaction.",
		HTTPStatus: 409,
	}
	ErrTerminalStateTransition = &DomainError{
		Code:       "terminal_state_error",
		Message:    "Transaction is in a terminal state and cannot transition to new states.",
		HTTPStatus: 409,
	}
	ErrInvalidStateTransition = &DomainError{
		Code:       "invalid_state_transition",
		Message:    "Requested state transition is not permitted by domain state machine.",
		HTTPStatus: 422,
	}
)

// Infrastructure / Acquirer Errors (502 / 503 / 504 / 202)
var (
	ErrAcquirerUnavailable = &DomainError{
		Code:       "acquirer_unavailable",
		Message:    "Acquirer network endpoint returned a temporary outage (503).",
		HTTPStatus: 503,
	}
	ErrAcquirerTimeout = &DomainError{
		Code:       "acquirer_timeout",
		Message:    "Acquirer request exceeded socket read/write execution timeout budget.",
		HTTPStatus: 504,
	}
	ErrAcquirerUnknownResult = &DomainError{
		Code:       "acquirer_unknown_result",
		Message:    "Acquirer authorization timed out and on-demand reconciliation could not confirm outcome.",
		HTTPStatus: 202,
	}
	ErrAcquirerUnexpected = &DomainError{
		Code:       "acquirer_unexpected_error",
		Message:    "Acquirer returned an unrecognized or malformed response payload.",
		HTTPStatus: 502,
	}
)
