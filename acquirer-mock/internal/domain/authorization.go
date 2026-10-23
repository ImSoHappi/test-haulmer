package domain

import (
	"fmt"
	"log/slog"
	"time"
)

// CardPayload contains raw card credentials sent by the payment service.
type CardPayload struct {
	Number      string `json:"number"`
	ExpiryMonth int    `json:"expiry_month"`
	ExpiryYear  int    `json:"expiry_year"`
	CVV         string `json:"cvv"`
	HolderName  string `json:"holder_name"`
}

// LogValue implements slog.LogValuer to prevent PAN and CVV leaks in logs.
func (c CardPayload) LogValue() slog.Value {
	maskedPAN := MaskPAN(c.Number)
	return slog.GroupValue(
		slog.String("masked_pan", maskedPAN),
		slog.Int("expiry_month", c.ExpiryMonth),
		slog.Int("expiry_year", c.ExpiryYear),
		slog.String("holder_name", c.HolderName),
	)
}

// String implements fmt.Stringer to prevent accidental plaintext interpolation.
func (c CardPayload) String() string {
	return fmt.Sprintf("CardPayload{PAN: %s, Holder: %s}", MaskPAN(c.Number), c.HolderName)
}

// MaskPAN returns a masked representation of a card number (BIN + ****** + Last4).
func MaskPAN(pan string) string {
	if len(pan) < 10 {
		return "******"
	}
	return pan[:6] + "******" + pan[len(pan)-4:]
}

// AuthorizationRequest represents an incoming authorization attempt.
type AuthorizationRequest struct {
	ReferenceID string      `json:"reference_id"`
	Amount      int64       `json:"amount"`
	Currency    string      `json:"currency"`
	Card        CardPayload `json:"card"`
}

// AuthorizationResponse represents the acquirer authorization outcome.
type AuthorizationResponse struct {
	ReferenceID       string    `json:"reference_id"`
	Status            string    `json:"status"`
	Code              string    `json:"code"`
	DeclineCode       string    `json:"decline_code,omitempty"`
	Message           string    `json:"message,omitempty"`
	AuthorizationCode string    `json:"authorization_code,omitempty"`
	Amount            int64     `json:"amount"`
	Currency          string    `json:"currency"`
	CreatedAt         time.Time `json:"created_at"`
}

// ErrorResponse represents an acquirer infrastructure or validation error.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}
