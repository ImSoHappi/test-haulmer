package acquirer

import (
	"context"
	"time"
)

// CardRequest holds the card payload sent to the acquirer.
// Note: In logs and runtime representation, sensitive credentials follow PCI-DSS rules.
type CardRequest struct {
	Number      string `json:"number"`
	ExpiryMonth int    `json:"expiry_month"`
	ExpiryYear  int    `json:"expiry_year"`
	CVV         string `json:"cvv"`
	HolderName  string `json:"holder_name"`
}

// AuthorizeRequest models the payload sent to POST /authorizations.
type AuthorizeRequest struct {
	ReferenceID   string      `json:"reference_id"`
	Amount        int64       `json:"amount"`
	Currency      string      `json:"currency"`
	Card          CardRequest `json:"card"`
	CorrelationID string      `json:"-"`
}

// AcquirerResponse models the response from POST /authorizations or GET /authorizations/{ref}.
type AcquirerResponse struct {
	ReferenceID       string    `json:"reference_id"`
	Status            string    `json:"status"` // APPROVED, DECLINED, FAILED
	Code              string    `json:"code"`   // 00, 51, acquirer_down, etc.
	AuthorizationCode string    `json:"authorization_code,omitempty"`
	Amount            int64     `json:"amount"`
	Currency          string    `json:"currency"`
	CreatedAt         time.Time `json:"created_at"`
}

// AcquirerPort defines the secondary port interface for communicating with the bank acquirer.
type AcquirerPort interface {
	Authorize(ctx context.Context, req AuthorizeRequest) (*AcquirerResponse, error)
	GetAuthorization(ctx context.Context, referenceID string, correlationID string) (*AcquirerResponse, error)
}
