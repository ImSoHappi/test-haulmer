package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

// CreateTransactionEvent appends an immutable audit log entry for a transaction.
func (r *Repository) CreateTransactionEvent(ctx context.Context, tx any, e *domain.TransactionEvent) error {
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
