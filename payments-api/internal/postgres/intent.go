package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

// CreatePaymentIntent persists a new PaymentIntent record.
func (r *Repository) CreatePaymentIntent(ctx context.Context, tx any, pi *domain.PaymentIntent) error {
	query := `
		INSERT INTO payment_intents (
			id, merchant_id, order_reference, amount, currency, metadata, correlation_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	metadataBytes, err := json.Marshal(pi.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.runner(tx).Exec(ctx, query,
		pi.ID,
		pi.MerchantID,
		pi.OrderReference,
		pi.Amount.Int64(),
		pi.Currency,
		metadataBytes,
		pi.CorrelationID,
		pi.CreatedAt,
		pi.UpdatedAt,
	)
	if err != nil {
		return mapPgError(err)
	}
	return nil
}

// GetPaymentIntentByOrderRef retrieves a PaymentIntent by merchant ID and order reference.
func (r *Repository) GetPaymentIntentByOrderRef(ctx context.Context, merchantID, orderRef string) (*domain.PaymentIntent, error) {
	query := `
		SELECT id, merchant_id, order_reference, amount, currency, metadata, correlation_id, created_at, updated_at
		FROM payment_intents
		WHERE merchant_id = $1 AND order_reference = $2
	`
	var (
		pi            domain.PaymentIntent
		rawAmount     int64
		metadataBytes []byte
	)

	err := r.pool.QueryRow(ctx, query, merchantID, orderRef).Scan(
		&pi.ID,
		&pi.MerchantID,
		&pi.OrderReference,
		&rawAmount,
		&pi.Currency,
		&metadataBytes,
		&pi.CorrelationID,
		&pi.CreatedAt,
		&pi.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query payment intent: %w", err)
	}

	amt, err := domain.NewAmount(rawAmount)
	if err != nil {
		return nil, fmt.Errorf("corrupt amount in database: %w", err)
	}
	pi.Amount = amt

	if len(metadataBytes) > 0 {
		_ = json.Unmarshal(metadataBytes, &pi.Metadata)
	}

	return &pi, nil
}
