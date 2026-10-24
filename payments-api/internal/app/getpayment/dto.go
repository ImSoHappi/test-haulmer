package getpayment

import "time"

// Request specifies criteria to retrieve a payment and its audit events.
type Request struct {
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

// Response represents the full details of a transaction and its audit history.
type Response struct {
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
	Events            []EventItem `json:"events"`
}
