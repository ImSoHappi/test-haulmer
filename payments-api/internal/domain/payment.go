package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TransactionStatus represents the finite states of a payment transaction.
// This is the SINGLE SOURCE OF TRUTH for payment status across the entire system.
type TransactionStatus string

const (
	StatusPending    TransactionStatus = "PENDING"
	StatusProcessing TransactionStatus = "PROCESSING"
	StatusApproved   TransactionStatus = "APPROVED"
	StatusDeclined   TransactionStatus = "DECLINED"
	StatusFailed     TransactionStatus = "FAILED"
)

// IsTerminal returns true if the status represents an unalterable final outcome.
func (s TransactionStatus) IsTerminal() bool {
	switch s {
	case StatusApproved, StatusDeclined, StatusFailed:
		return true
	default:
		return false
	}
}

// CanTransition validates whether transitioning from one status to another is legal.
func CanTransition(from, to TransactionStatus) bool {
	if from.IsTerminal() {
		return false
	}

	switch from {
	case StatusPending:
		return to == StatusProcessing || to == StatusDeclined || to == StatusFailed
	case StatusProcessing:
		return to == StatusApproved || to == StatusDeclined || to == StatusFailed
	default:
		return false
	}
}

// DeclineSource identifies whether the transaction was declined internally or by the acquirer.
type DeclineSource string

const (
	DeclineSourceInternal DeclineSource = "INTERNAL"
	DeclineSourceAcquirer DeclineSource = "ACQUIRER"
)

// PaymentIntent represents the merchant's high-level payment intention.
type PaymentIntent struct {
	ID             uuid.UUID      `json:"id"`
	MerchantID     string         `json:"merchant_id"`
	OrderReference string         `json:"order_reference"`
	Amount         Amount         `json:"amount"`
	Currency       string         `json:"currency"`
	Metadata       map[string]any `json:"metadata"`
	CorrelationID  string         `json:"correlation_id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// NewPaymentIntent constructs a new validated PaymentIntent.
func NewPaymentIntent(
	id uuid.UUID,
	merchantID string,
	orderRef string,
	amount Amount,
	currency string,
	metadata map[string]any,
	correlationID string,
	now time.Time,
) (*PaymentIntent, error) {
	if merchantID == "" || orderRef == "" || correlationID == "" {
		return nil, ErrMissingRequiredField
	}
	if metadata == nil {
		metadata = make(map[string]any)
	}

	return &PaymentIntent{
		ID:             id,
		MerchantID:     merchantID,
		OrderReference: orderRef,
		Amount:         amount,
		Currency:       currency,
		Metadata:       metadata,
		CorrelationID:  correlationID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// Transaction represents a concrete authorization attempt against the acquirer.
type Transaction struct {
	ID                uuid.UUID         `json:"id"`
	PaymentIntentID   uuid.UUID         `json:"payment_intent_id"`
	MerchantID        string            `json:"merchant_id"`
	IdempotencyKey    string            `json:"idempotency_key"`
	PayloadHash       string            `json:"payload_hash"`
	Status            TransactionStatus `json:"status"`
	CardBIN           string            `json:"card_bin"`
	CardLast4         string            `json:"card_last4"`
	DeclineSource     DeclineSource     `json:"decline_source,omitempty"`
	ResultCode        string            `json:"result_code,omitempty"`
	ResultMessage     string            `json:"result_message,omitempty"`
	AuthorizationCode string            `json:"authorization_code,omitempty"`
	CorrelationID     string            `json:"correlation_id"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	CompletedAt       *time.Time        `json:"completed_at,omitempty"`
}

// NewTransaction constructs a new transaction in PENDING status.
func NewTransaction(
	id uuid.UUID,
	paymentIntentID uuid.UUID,
	merchantID string,
	idempotencyKey string,
	payloadHash string,
	cardBIN string,
	cardLast4 string,
	correlationID string,
	now time.Time,
) *Transaction {
	return &Transaction{
		ID:              id,
		PaymentIntentID: paymentIntentID,
		MerchantID:      merchantID,
		IdempotencyKey:  idempotencyKey,
		PayloadHash:     payloadHash,
		Status:          StatusPending,
		CardBIN:         cardBIN,
		CardLast4:       cardLast4,
		CorrelationID:   correlationID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// TransitionTo validates and applies a status transition.
func (t *Transaction) TransitionTo(target TransactionStatus, now time.Time) error {
	if t.Status.IsTerminal() {
		return ErrTerminalStateTransition
	}
	if !CanTransition(t.Status, target) {
		return ErrInvalidStateTransition
	}

	t.Status = target
	t.UpdatedAt = now
	return nil
}

// MarkApproved transitions the transaction to APPROVED status and sets approval details.
func (t *Transaction) MarkApproved(authCode, code, msg string, completedAt time.Time) error {
	if err := t.TransitionTo(StatusApproved, completedAt); err != nil {
		return err
	}
	t.AuthorizationCode = authCode
	t.ResultCode = code
	t.ResultMessage = msg
	t.CompletedAt = &completedAt
	return nil
}

// MarkDeclined transitions the transaction to DECLINED status and sets decline details.
func (t *Transaction) MarkDeclined(source DeclineSource, code, msg string, completedAt time.Time) error {
	if err := t.TransitionTo(StatusDeclined, completedAt); err != nil {
		return err
	}
	t.DeclineSource = source
	t.ResultCode = code
	t.ResultMessage = msg
	t.CompletedAt = &completedAt
	return nil
}

// MarkFailed transitions the transaction to FAILED status due to infrastructure/timeout errors.
func (t *Transaction) MarkFailed(code, msg string, completedAt time.Time) error {
	if err := t.TransitionTo(StatusFailed, completedAt); err != nil {
		return err
	}
	t.ResultCode = code
	t.ResultMessage = msg
	t.CompletedAt = &completedAt
	return nil
}

// EventType represents the category of an audit transaction event.
type EventType string

const (
	EventTransactionCreated EventType = "TRANSACTION_CREATED"
	EventStatusChanged      EventType = "STATUS_CHANGED"
	EventAcquirerAttempt    EventType = "ACQUIRER_ATTEMPT"
)

// TransactionEvent represents an immutable audit log entry for a transaction.
type TransactionEvent struct {
	ID              int64              `json:"id"`
	TransactionID   uuid.UUID          `json:"transaction_id"`
	EventType       EventType          `json:"event_type"`
	FromStatus      *TransactionStatus `json:"from_status,omitempty"`
	ToStatus        *TransactionStatus `json:"to_status,omitempty"`
	AcquirerAttempt *int               `json:"acquirer_attempt,omitempty"`
	LatencyMs       *int               `json:"latency_ms,omitempty"`
	Code            string             `json:"code,omitempty"`
	Message         string             `json:"message,omitempty"`
	Details         map[string]any     `json:"details,omitempty"`
	CorrelationID   string             `json:"correlation_id"`
	OccurredAt      time.Time          `json:"occurred_at"`
}

// ComputePayloadHash calculates a deterministic SHA-256 hash across canonical payment parameters.
// PCI Note: Sensitive credentials (full PAN, CVV) are NEVER included in the hash.
func ComputePayloadHash(merchantID, orderRef string, amount int64, currency, cardBIN, cardLast4 string, expMonth, expYear int) string {
	canonical := fmt.Sprintf("%s|%s|%d|%s|%s|%s|%02d/%04d",
		merchantID, orderRef, amount, currency, cardBIN, cardLast4, expMonth, expYear,
	)
	hash := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(hash[:])
}
