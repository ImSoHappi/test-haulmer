package processpayment

import "time"

// Request encapsulates all parameters required to charge a card.
type Request struct {
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

// Response represents the resulting authorization details.
type Response struct {
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
