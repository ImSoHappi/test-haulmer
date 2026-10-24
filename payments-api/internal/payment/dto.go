package payment

import (
	"time"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

// ProcessRequest encapsulates all parameters required to charge a card.
type ProcessRequest struct {
	MerchantID      string
	IdempotencyKey  string
	OrderReference  string
	Amount          int64
	Currency        string
	CardNumber      string
	CardExpiryMonth int
	CardExpiryYear  int
	CardCVV         string
	CardHolderName  string
	CorrelationID   string
}

// ProcessResponse represents the resulting authorization details.
type ProcessResponse struct {
	TransactionID     string     `json:"transaction_id"`
	PaymentIntentID   string     `json:"payment_intent_id"`
	MerchantID        string     `json:"merchant_id"`
	OrderReference    string     `json:"order_reference"`
	Amount            int64      `json:"amount"`
	Currency          string     `json:"currency"`
	Status            string     `json:"status"`
	CardBIN           string     `json:"card_bin"`
	CardLast4         string     `json:"card_last4"`
	AuthorizationCode string     `json:"authorization_code,omitempty"`
	DeclineSource     string     `json:"decline_source,omitempty"`
	ResultCode        string     `json:"result_code,omitempty"`
	ResultMessage     string     `json:"result_message,omitempty"`
	IsReplayed        bool       `json:"is_replayed"`
	CreatedAt         time.Time  `json:"created_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
}

// GetRequest specifies criteria to retrieve a payment and its audit events.
type GetRequest struct {
	MerchantID    string
	TransactionID string
}

// EventItem represents an individual audit event in the timeline.
type EventItem struct {
	ID        string         `json:"id"`
	EventType string         `json:"event_type"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}

// GetResponse represents the full details of a transaction and its audit history.
type GetResponse struct {
	ID                string      `json:"id"`
	PaymentIntentID   string      `json:"payment_intent_id"`
	MerchantID        string      `json:"merchant_id"`
	IdempotencyKey    string      `json:"idempotency_key"`
	Status            string      `json:"status"`
	CardBIN           string      `json:"card_bin"`
	CardLast4         string      `json:"card_last4"`
	DeclineSource     string      `json:"decline_source,omitempty"`
	ResultCode        string      `json:"result_code,omitempty"`
	ResultMessage     string      `json:"result_message,omitempty"`
	AuthorizationCode string      `json:"authorization_code,omitempty"`
	CorrelationID     string      `json:"correlation_id"`
	OrderReference    string      `json:"order_reference"`
	Amount            int64       `json:"amount"`
	Currency          string      `json:"currency"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
	CompletedAt       *time.Time  `json:"completed_at,omitempty"`
	Events            []EventItem `json:"events"`
}

// ListRequest specifies criteria for listing and filtering merchant transactions.
type ListRequest struct {
	MerchantID string
	Status     *domain.TransactionStatus
	Limit      int
	Offset     int
}

// ListItem represents an individual transaction item in the list response.
type ListItem struct {
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

// ListResponse represents the paginated result set of transactions.
type ListResponse struct {
	Items      []ListItem `json:"items"`
	TotalCount int        `json:"total_count"`
	Limit      int        `json:"limit"`
	Offset     int        `json:"offset"`
}

// Filter specifies search and pagination criteria for querying transactions in persistence.
type Filter struct {
	MerchantID *string
	Status     *domain.TransactionStatus
	Limit      int
	Offset     int
}
