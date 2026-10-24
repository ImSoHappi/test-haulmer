package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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
func (r *Repository) WithTx(ctx context.Context, fn func(tx any) error) error {
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

func (r *Repository) runner(tx any) QueryRunner {
	if qr, ok := tx.(QueryRunner); ok && qr != nil {
		return qr
	}
	return r.pool
}
