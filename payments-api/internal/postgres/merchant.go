package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

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
