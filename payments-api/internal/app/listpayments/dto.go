package listpayments

import (
	"time"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

// Request specifies criteria for listing and filtering merchant transactions.
type Request struct {
	MerchantID string
	Status     *domain.TransactionStatus
	Limit      int
	Offset     int
}

// Item represents an individual transaction item in the list response.
type Item struct {
	ID                string     `json:"id"`
	PaymentIntentID   string     `json:"payment_intent_id"`
	MerchantID        string     `json:"merchant_id"`
	IdempotencyKey    string     `json:"idempotency_key"`
	Status            string     `json:"status"`
	CardBIN           string     `json:"card_bin"`
	CardLast4         string     `json:"card_last4"`
	DeclineSource     string     `json:"decline_source,omitempty"`
	ResultCode        string     `json:"result_code,omitempty"`
	ResultMessage     string     `json:"result_message,omitempty"`
	AuthorizationCode string     `json:"authorization_code,omitempty"`
	CorrelationID     string     `json:"correlation_id"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
}

// Response represents the paginated result set of transactions.
type Response struct {
	Items      []Item `json:"items"`
	TotalCount int    `json:"total_count"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}

// Filter specifies search and pagination criteria for querying transactions in persistence.
type Filter struct {
	MerchantID *string
	Status     *domain.TransactionStatus
	Limit      int
	Offset     int
}
