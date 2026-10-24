package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

// QueryRunner defines the common interface implemented by *pgxpool.Pool and pgx.Tx.
type QueryRunner interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Repository implements the secondary database adapter for PostgreSQL using pgx/v5.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a Repository wrapping an active pgxpool.Pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// WithTx executes the provided callback within an atomic database transaction.
// It commits on success and rolls back on failure or panic.
func (r *Repository) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *Repository) runner(tx QueryRunner) QueryRunner {
	if tx != nil {
		return tx
	}
	return r.pool
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

// --- Merchant Operations ---

// GetMerchantByID retrieves a merchant by ID. Returns ErrMerchantNotFound if not present.
func (r *Repository) GetMerchantByID(ctx context.Context, id string) (*domain.Merchant, error) {
	query := `SELECT id, name, status FROM merchants WHERE id = $1`
	var m domain.Merchant
	err := r.pool.QueryRow(ctx, query, id).Scan(&m.ID, &m.Name, &m.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMerchantNotFound
		}
		return nil, fmt.Errorf("failed to query merchant: %w", err)
	}
	return &m, nil
}

// --- PaymentIntent Operations ---

// CreatePaymentIntent persists a new PaymentIntent record.
func (r *Repository) CreatePaymentIntent(ctx context.Context, tx QueryRunner, pi *domain.PaymentIntent) error {
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

// --- Transaction Operations ---

// CreateTransaction inserts a new transaction. Translates PostgreSQL unique constraint violations (23505)
// to semantic domain errors (ErrIdempotencyConflict, ErrIntentAlreadyApproved, ErrActiveTransactionExists).
func (r *Repository) CreateTransaction(ctx context.Context, tx QueryRunner, t *domain.Transaction) error {
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
func (r *Repository) UpdateTransactionStatus(ctx context.Context, tx QueryRunner, t *domain.Transaction, fromStatus domain.TransactionStatus) (bool, error) {
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

// ListFilter specifies parameters for filtering and paginating transactions.
type ListFilter struct {
	MerchantID *string
	Status     *domain.TransactionStatus
	Limit      int
	Offset     int
}

// ListTransactions retrieves transactions matching the optional filter criteria, ordered by created_at DESC.
func (r *Repository) ListTransactions(ctx context.Context, filter ListFilter) ([]*domain.Transaction, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
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

// --- TransactionEvent Operations ---

// CreateTransactionEvent appends an immutable audit log entry for a transaction.
func (r *Repository) CreateTransactionEvent(ctx context.Context, tx QueryRunner, e *domain.TransactionEvent) error {
	query := `
		INSERT INTO transaction_events (
			transaction_id, event_type, from_status, to_status, acquirer_attempt,
			latency_ms, code, message, details, correlation_id, occurred_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	var (
		fromStatusStr *string
		toStatusStr   *string
		detailsBytes  []byte
	)
	if e.FromStatus != nil {
		s := string(*e.FromStatus)
		fromStatusStr = &s
	}
	if e.ToStatus != nil {
		s := string(*e.ToStatus)
		toStatusStr = &s
	}
	if len(e.Details) > 0 {
		var err error
		detailsBytes, err = json.Marshal(e.Details)
		if err != nil {
			return fmt.Errorf("failed to marshal event details: %w", err)
		}
	} else {
		detailsBytes = []byte("{}")
	}

	_, err := r.runner(tx).Exec(ctx, query,
		e.TransactionID,
		string(e.EventType),
		fromStatusStr,
		toStatusStr,
		e.AcquirerAttempt,
		e.LatencyMs,
		e.Code,
		e.Message,
		detailsBytes,
		e.CorrelationID,
		e.OccurredAt,
	)
	if err != nil {
		return mapPgError(err)
	}
	return nil
}

// ListEventsByTransactionID retrieves all audit events for a transaction ordered chronologically by ID.
func (r *Repository) ListEventsByTransactionID(ctx context.Context, txID uuid.UUID) ([]*domain.TransactionEvent, error) {
	query := `
		SELECT id, transaction_id, event_type, from_status, to_status, acquirer_attempt,
		       latency_ms, code, message, details, correlation_id, occurred_at
		FROM transaction_events
		WHERE transaction_id = $1
		ORDER BY id ASC
	`
	rows, err := r.pool.Query(ctx, query, txID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction events: %w", err)
	}
	defer rows.Close()

	var events []*domain.TransactionEvent
	for rows.Next() {
		var (
			e             domain.TransactionEvent
			eventTypeStr  string
			fromStatusStr *string
			toStatusStr   *string
			detailsBytes  []byte
		)
		err := rows.Scan(
			&e.ID,
			&e.TransactionID,
			&eventTypeStr,
			&fromStatusStr,
			&toStatusStr,
			&e.AcquirerAttempt,
			&e.LatencyMs,
			&e.Code,
			&e.Message,
			&detailsBytes,
			&e.CorrelationID,
			&e.OccurredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction event row: %w", err)
		}

		e.EventType = domain.EventType(eventTypeStr)
		if fromStatusStr != nil {
			st := domain.TransactionStatus(*fromStatusStr)
			e.FromStatus = &st
		}
		if toStatusStr != nil {
			st := domain.TransactionStatus(*toStatusStr)
			e.ToStatus = &st
		}
		if len(detailsBytes) > 0 {
			_ = json.Unmarshal(detailsBytes, &e.Details)
		}
		events = append(events, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading event rows: %w", err)
	}

	return events, nil
}

// mapPgError maps PostgreSQL unique violation errors (23505) to specific domain errors.
func mapPgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "uq_tx_idempotency":
				return domain.ErrIdempotencyConflict
			case "uq_tx_one_approved":
				return domain.ErrIntentAlreadyApproved
			case "uq_tx_one_active":
				return domain.ErrActiveTransactionExists
			}
		}
	}
	return err
}
