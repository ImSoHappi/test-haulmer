package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

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
