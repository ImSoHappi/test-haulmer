package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/payment"
)

// ComputePayloadHash calculates a deterministic SHA-256 hash across canonical payment parameters.
// PCI Note: Sensitive credentials (full PAN, CVV) are NEVER included in the hash.
func ComputePayloadHash(merchantID, orderRef string, amount int64, currency, cardBIN, cardLast4 string, expMonth, expYear int) string {
	return domain.ComputePayloadHash(merchantID, orderRef, amount, currency, cardBIN, cardLast4, expMonth, expYear)
}

// CreateTransaction inserts a new transaction. Translates PostgreSQL unique constraint violations (23505)
// to semantic domain errors (ErrIdempotencyConflict, ErrIntentAlreadyApproved, ErrActiveTransactionExists).
func (r *Repository) CreateTransaction(ctx context.Context, tx any, t *domain.Transaction) error {
	query := `
		INSERT INTO transactions (
			id, payment_intent_id, merchant_id, idempotency_key, payload_hash, status,
			card_bin, card_last4, correlation_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.runner(tx).Exec(ctx, query,
		t.ID,
		t.PaymentIntentID,
		t.MerchantID,
		t.IdempotencyKey,
		t.PayloadHash,
		string(t.Status),
		t.CardBIN,
		t.CardLast4,
		t.CorrelationID,
		t.CreatedAt,
		t.UpdatedAt,
	)
	if err != nil {
		return mapPgError(err)
	}
	return nil
}

// UpdateTransactionStatus updates transaction outcome using optimistic concurrency check (WHERE id = $1 AND status = $2).
// Returns (true, nil) if row was updated, or (false, nil) if status was already changed concurrently.
func (r *Repository) UpdateTransactionStatus(ctx context.Context, tx any, t *domain.Transaction, fromStatus domain.TransactionStatus) (bool, error) {
	query := `
		UPDATE transactions
		SET status = $1,
		    decline_source = NULLIF($2, '')::text,
		    result_code = NULLIF($3, '')::text,
		    result_message = NULLIF($4, '')::text,
		    authorization_code = NULLIF($5, '')::text,
		    updated_at = $6,
		    completed_at = $7
		WHERE id = $8 AND status = $9
	`
	var declineSourceStr string
	if t.DeclineSource != "" {
		declineSourceStr = string(t.DeclineSource)
	}

	tag, err := r.runner(tx).Exec(ctx, query,
		string(t.Status),
		declineSourceStr,
		t.ResultCode,
		t.ResultMessage,
		t.AuthorizationCode,
		t.UpdatedAt,
		t.CompletedAt,
		t.ID,
		string(fromStatus),
	)
	if err != nil {
		return false, mapPgError(err)
	}

	return tag.RowsAffected() > 0, nil
}

// GetTransactionByIdempotencyKey retrieves a transaction by merchant ID and idempotency key.
func (r *Repository) GetTransactionByIdempotencyKey(ctx context.Context, merchantID, idempotencyKey string) (*domain.Transaction, error) {
	query := `
		SELECT id, payment_intent_id, merchant_id, idempotency_key, payload_hash, status,
		       card_bin, card_last4, COALESCE(decline_source, ''), COALESCE(result_code, ''),
		       COALESCE(result_message, ''), COALESCE(authorization_code, ''),
		       correlation_id, created_at, updated_at, completed_at
		FROM transactions
		WHERE merchant_id = $1 AND idempotency_key = $2
	`
	t, err := r.scanTransaction(r.pool.QueryRow(ctx, query, merchantID, idempotencyKey))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query transaction by idempotency key: %w", err)
	}
	return t, nil
}

// GetTransactionByID retrieves a transaction by its primary key ID.
func (r *Repository) GetTransactionByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	query := `
		SELECT id, payment_intent_id, merchant_id, idempotency_key, payload_hash, status,
		       card_bin, card_last4, COALESCE(decline_source, ''), COALESCE(result_code, ''),
		       COALESCE(result_message, ''), COALESCE(authorization_code, ''),
		       correlation_id, created_at, updated_at, completed_at
		FROM transactions
		WHERE id = $1
	`
	t, err := r.scanTransaction(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query transaction by ID: %w", err)
	}
	return t, nil
}

// ListTransactions retrieves transactions matching the optional filter criteria, ordered by created_at DESC.
func (r *Repository) ListTransactions(ctx context.Context, filter payment.Filter) ([]*domain.Transaction, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 50 {
		filter.Limit = 50
	}

	query := `
		SELECT id, payment_intent_id, merchant_id, idempotency_key, payload_hash, status,
		       card_bin, card_last4, COALESCE(decline_source, ''), COALESCE(result_code, ''),
		       COALESCE(result_message, ''), COALESCE(authorization_code, ''),
		       correlation_id, created_at, updated_at, completed_at,
		       COUNT(*) OVER() AS total_count
		FROM transactions
		WHERE ($1::text IS NULL OR merchant_id = $1)
		  AND ($2::text IS NULL OR status = $2)
		ORDER BY created_at DESC, id DESC
		LIMIT $3 OFFSET $4
	`
	var (
		merchantArg any
		statusArg   any
	)
	if filter.MerchantID != nil {
		merchantArg = *filter.MerchantID
	}
	if filter.Status != nil {
		statusArg = string(*filter.Status)
	}

	rows, err := r.pool.Query(ctx, query, merchantArg, statusArg, filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list transactions: %w", err)
	}
	defer rows.Close()

	var (
		list       []*domain.Transaction
		totalCount int
	)

	for rows.Next() {
		var (
			t             domain.Transaction
			statusStr     string
			declineSource string
		)
		err := rows.Scan(
			&t.ID,
			&t.PaymentIntentID,
			&t.MerchantID,
			&t.IdempotencyKey,
			&t.PayloadHash,
			&statusStr,
			&t.CardBIN,
			&t.CardLast4,
			&declineSource,
			&t.ResultCode,
			&t.ResultMessage,
			&t.AuthorizationCode,
			&t.CorrelationID,
			&t.CreatedAt,
			&t.UpdatedAt,
			&t.CompletedAt,
			&totalCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan transaction row: %w", err)
		}
		t.Status = domain.TransactionStatus(statusStr)
		if declineSource != "" {
			t.DeclineSource = domain.DeclineSource(declineSource)
		}
		list = append(list, &t)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error reading transaction rows: %w", err)
	}

	return list, totalCount, nil
}

func (r *Repository) scanTransaction(row pgx.Row) (*domain.Transaction, error) {
	var (
		t             domain.Transaction
		statusStr     string
		declineSource string
	)
	err := row.Scan(
		&t.ID,
		&t.PaymentIntentID,
		&t.MerchantID,
		&t.IdempotencyKey,
		&t.PayloadHash,
		&statusStr,
		&t.CardBIN,
		&t.CardLast4,
		&declineSource,
		&t.ResultCode,
		&t.ResultMessage,
		&t.AuthorizationCode,
		&t.CorrelationID,
		&t.CreatedAt,
		&t.UpdatedAt,
		&t.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	t.Status = domain.TransactionStatus(statusStr)
	if declineSource != "" {
		t.DeclineSource = domain.DeclineSource(declineSource)
	}
	return &t, nil
}
