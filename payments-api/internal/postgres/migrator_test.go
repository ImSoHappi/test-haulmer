package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/postgres"
)

func getTestDatabaseURL() string {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		return url
	}
	return "postgres://postgres:postgres@localhost:5432/payments_test?sslmode=disable"
}

func TestMigrator_LifecycleAndConstraints(t *testing.T) {
	dbURL := getTestDatabaseURL()
	ctx := context.Background()

	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err, "failed to open database connection")
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		t.Skipf("skipping postgres integration test: database unreachable at %s: %v", dbURL, err)
	}

	// Step 1: Ensure clean state before running
	_ = postgres.RollbackAll(dbURL)

	// Step 2: Apply all migrations up
	t.Run("ApplyMigrations_Up", func(t *testing.T) {
		err := postgres.RunMigrations(dbURL)
		require.NoError(t, err, "RunMigrations should succeed")

		// Verify that all 4 tables exist in information_schema
		tables := []string{"merchants", "payment_intents", "transactions", "transaction_events"}
		for _, table := range tables {
			var exists bool
			query := `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1);`
			err := db.QueryRowContext(ctx, query, table).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "table %s should exist in database", table)
		}
	})

	// Step 3: Verify Seeded Merchants
	t.Run("Verify_SeededMerchants", func(t *testing.T) {
		var activeName, activeStatus string
		err := db.QueryRowContext(ctx, "SELECT name, status FROM merchants WHERE id = '1001'").Scan(&activeName, &activeStatus)
		require.NoError(t, err)
		assert.Equal(t, "Comercio Activo Principal", activeName)
		assert.Equal(t, "ACTIVE", activeStatus)

		var inactiveStatus string
		err = db.QueryRowContext(ctx, "SELECT status FROM merchants WHERE id = '1002'").Scan(&inactiveStatus)
		require.NoError(t, err)
		assert.Equal(t, "INACTIVE", inactiveStatus)
	})

	// Step 4: Verify Idempotency Constraint (uq_tx_idempotency)
	t.Run("Constraint_Idempotency_Collision", func(t *testing.T) {
		intentID := uuid.New()
		_, err := db.ExecContext(ctx, `
			INSERT INTO payment_intents (id, merchant_id, order_reference, amount, currency, metadata, correlation_id)
			VALUES ($1, '1001', 'ORD-TEST-001', 10000, 'CLP', '{}', 'corr-1');
		`, intentID)
		require.NoError(t, err)

		txID1 := uuid.New()
		_, err = db.ExecContext(ctx, `
			INSERT INTO transactions (id, payment_intent_id, merchant_id, idempotency_key, payload_hash, status, card_bin, card_last4, correlation_id)
			VALUES ($1, $2, '1001', 'idemp-key-123', 'hash123', 'PROCESSING', '411111', '1111', 'corr-1');
		`, txID1, intentID)
		require.NoError(t, err)

		// Second insert with SAME merchant_id and SAME idempotency_key
		txID2 := uuid.New()
		_, err = db.ExecContext(ctx, `
			INSERT INTO transactions (id, payment_intent_id, merchant_id, idempotency_key, payload_hash, status, card_bin, card_last4, correlation_id)
			VALUES ($1, $2, '1001', 'idemp-key-123', 'hash123', 'PROCESSING', '411111', '1111', 'corr-2');
		`, txID2, intentID)
		require.Error(t, err, "should fail on duplicate idempotency key")

		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			assert.Equal(t, "23505", string(pqErr.Code), "should be a unique violation error (23505)")
			assert.Equal(t, "uq_tx_idempotency", pqErr.Constraint, "should trigger uq_tx_idempotency constraint")
		} else {
			t.Fatalf("expected *pq.Error, got %T: %v", err, err)
		}
	})

	// Step 5: Verify Single Approved Constraint (uq_tx_one_approved)
	t.Run("Constraint_SingleApprovedPerIntent", func(t *testing.T) {
		intentID := uuid.New()
		_, err := db.ExecContext(ctx, `
			INSERT INTO payment_intents (id, merchant_id, order_reference, amount, currency, metadata, correlation_id)
			VALUES ($1, '1001', 'ORD-TEST-002', 15000, 'CLP', '{}', 'corr-app-1');
		`, intentID)
		require.NoError(t, err)

		// Insert first APPROVED transaction
		txID1 := uuid.New()
		_, err = db.ExecContext(ctx, `
			INSERT INTO transactions (id, payment_intent_id, merchant_id, idempotency_key, payload_hash, status, card_bin, card_last4, correlation_id)
			VALUES ($1, $2, '1001', 'idemp-app-1', 'hash-app-1', 'APPROVED', '411111', '1111', 'corr-app-1');
		`, txID1, intentID)
		require.NoError(t, err)

		// Attempt to insert second APPROVED transaction for the SAME payment intent
		txID2 := uuid.New()
		_, err = db.ExecContext(ctx, `
			INSERT INTO transactions (id, payment_intent_id, merchant_id, idempotency_key, payload_hash, status, card_bin, card_last4, correlation_id)
			VALUES ($1, $2, '1001', 'idemp-app-2', 'hash-app-2', 'APPROVED', '411111', '2222', 'corr-app-2');
		`, txID2, intentID)
		require.Error(t, err, "should fail on second approved transaction for the same intent")

		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			assert.Equal(t, "23505", string(pqErr.Code))
			assert.Equal(t, "uq_tx_one_approved", pqErr.Constraint)
		} else {
			t.Fatalf("expected *pq.Error, got %T: %v", err, err)
		}
	})

	// Step 6: Verify Single Active Constraint (uq_tx_one_active)
	t.Run("Constraint_SingleActivePerIntent", func(t *testing.T) {
		intentID := uuid.New()
		_, err := db.ExecContext(ctx, `
			INSERT INTO payment_intents (id, merchant_id, order_reference, amount, currency, metadata, correlation_id)
			VALUES ($1, '1001', 'ORD-TEST-003', 20000, 'CLP', '{}', 'corr-act-1');
		`, intentID)
		require.NoError(t, err)

		// Insert first PROCESSING transaction
		txID1 := uuid.New()
		_, err = db.ExecContext(ctx, `
			INSERT INTO transactions (id, payment_intent_id, merchant_id, idempotency_key, payload_hash, status, card_bin, card_last4, correlation_id)
			VALUES ($1, $2, '1001', 'idemp-act-1', 'hash-act-1', 'PROCESSING', '411111', '1111', 'corr-act-1');
		`, txID1, intentID)
		require.NoError(t, err)

		// Attempt to insert concurrent second PROCESSING transaction for the same intent
		txID2 := uuid.New()
		_, err = db.ExecContext(ctx, `
			INSERT INTO transactions (id, payment_intent_id, merchant_id, idempotency_key, payload_hash, status, card_bin, card_last4, correlation_id)
			VALUES ($1, $2, '1001', 'idemp-act-2', 'hash-act-2', 'PROCESSING', '411111', '2222', 'corr-act-2');
		`, txID2, intentID)
		require.Error(t, err, "should fail on second concurrent active transaction")

		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			assert.Equal(t, "23505", string(pqErr.Code))
			assert.Equal(t, "uq_tx_one_active", pqErr.Constraint)
		} else {
			t.Fatalf("expected *pq.Error, got %T: %v", err, err)
		}
	})

	// Step 7: Verify RollbackAll (Down migrations)
	t.Run("RollbackAll_Down", func(t *testing.T) {
		err := postgres.RollbackAll(dbURL)
		require.NoError(t, err, "RollbackAll should cleanly drop tables")

		// Verify tables are removed
		tables := []string{"transaction_events", "transactions", "payment_intents", "merchants"}
		for _, table := range tables {
			var exists bool
			query := `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1);`
			err := db.QueryRowContext(ctx, query, table).Scan(&exists)
			require.NoError(t, err)
			assert.False(t, exists, "table %s should have been dropped", table)
		}

		// Re-apply so the database stays in a ready state
		err = postgres.RunMigrations(dbURL)
		require.NoError(t, err, "re-applying migrations should succeed")
	})
}
