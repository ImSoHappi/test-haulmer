package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/app/listpayments"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/postgres"
)

func setupTestDB(t *testing.T) (*postgres.Repository, func()) {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/payments_test?sslmode=disable"
	}

	ctx := context.Background()

	// 1. Ensure migrations are up to date
	err := postgres.RunMigrations(dbURL)
	require.NoError(t, err, "failed to apply migrations on test database")

	// 2. Initialize pgxpool
	pool, err := postgres.NewPool(ctx, dbURL)
	require.NoError(t, err, "failed to connect to test database pool")

	repo := postgres.NewRepository(pool)

	cleanup := func() {
		pool.Close()
	}

	return repo, cleanup
}

func TestRepository_GetMerchantByID(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Case 1: Active Merchant (m_1001)
	mActive, err := repo.GetMerchantByID(ctx, "m_1001")
	require.NoError(t, err)
	assert.Equal(t, "m_1001", mActive.ID)
	assert.Equal(t, domain.MerchantStatusActive, mActive.Status)
	assert.True(t, mActive.IsActive())

	// Case 2: Inactive Merchant (m_1002)
	mInactive, err := repo.GetMerchantByID(ctx, "m_1002")
	require.NoError(t, err)
	assert.Equal(t, "m_1002", mInactive.ID)
	assert.Equal(t, domain.MerchantStatusInactive, mInactive.Status)
	assert.False(t, mInactive.IsActive())

	// Case 3: Non-existent Merchant
	_, err = repo.GetMerchantByID(ctx, "m_non_existent")
	assert.ErrorIs(t, err, domain.ErrMerchantNotFound)
}

func TestRepository_AtomicPaymentFlow(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now().UTC()
	intentID := uuid.New()
	txID := uuid.New()
	orderRef := "ord-" + uuid.New().String()[:8]
	idemKey := "idem-" + uuid.New().String()[:8]
	corrID := "corr-" + uuid.New().String()[:8]

	amount, err := domain.NewAmount(25990)
	require.NoError(t, err)

	intent, err := domain.NewPaymentIntent(
		intentID,
		"m_1001",
		orderRef,
		amount,
		domain.CurrencyCLP,
		map[string]any{"customer_id": "cust_123"},
		corrID,
		now,
	)
	require.NoError(t, err)

	payloadHash := postgres.ComputePayloadHash("m_1001", orderRef, 25990, "CLP", "424242", "4242", 12, 2028)
	txRecord := domain.NewTransaction(
		txID,
		intentID,
		"m_1001",
		idemKey,
		payloadHash,
		"424242",
		"4242",
		corrID,
		now,
	)

	fromStatus := domain.StatusPending
	toStatus := domain.StatusProcessing
	event := &domain.TransactionEvent{
		TransactionID: txID,
		EventType:     domain.EventTransactionCreated,
		FromStatus:    &fromStatus,
		ToStatus:      &toStatus,
		CorrelationID: corrID,
		OccurredAt:    now,
	}

	// Execute atomic multi-statement insertion using WithTx
	err = repo.WithTx(ctx, func(tx any) error {
		if err := repo.CreatePaymentIntent(ctx, tx, intent); err != nil {
			return err
		}
		if err := repo.CreateTransaction(ctx, tx, txRecord); err != nil {
			return err
		}
		return repo.CreateTransactionEvent(ctx, tx, event)
	})
	require.NoError(t, err)

	// Verify reads outside the transaction
	foundIntent, err := repo.GetPaymentIntentByOrderRef(ctx, "m_1001", orderRef)
	require.NoError(t, err)
	assert.Equal(t, intentID, foundIntent.ID)
	assert.Equal(t, int64(25990), foundIntent.Amount.Int64())

	foundTx, err := repo.GetTransactionByID(ctx, txID)
	require.NoError(t, err)
	assert.Equal(t, txID, foundTx.ID)
	assert.Equal(t, domain.StatusPending, foundTx.Status)
	assert.Equal(t, payloadHash, foundTx.PayloadHash)

	events, err := repo.ListEventsByTransactionID(ctx, txID)
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, domain.EventTransactionCreated, events[0].EventType)
}

func TestRepository_ConstraintMappings_23505(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now().UTC()
	intentID := uuid.New()
	orderRef := "ord-constraint-" + uuid.New().String()[:8]
	idemKey := "idem-constraint-" + uuid.New().String()[:8]
	corrID := "corr-constraint-1"

	amount, _ := domain.NewAmount(15000)
	intent, _ := domain.NewPaymentIntent(intentID, "m_1001", orderRef, amount, domain.CurrencyCLP, nil, corrID, now)
	err := repo.CreatePaymentIntent(ctx, nil, intent)
	require.NoError(t, err)

	tx1 := domain.NewTransaction(uuid.New(), intentID, "m_1001", idemKey, "hash-1", "424242", "4242", corrID, now)
	err = repo.CreateTransaction(ctx, nil, tx1)
	require.NoError(t, err)

	// Test 1: Idempotency conflict (uq_tx_idempotency)
	txDuplicateIdem := domain.NewTransaction(uuid.New(), intentID, "m_1001", idemKey, "hash-2", "424242", "4242", corrID, now)
	err = repo.CreateTransaction(ctx, nil, txDuplicateIdem)
	assert.ErrorIs(t, err, domain.ErrIdempotencyConflict, "must map uq_tx_idempotency to ErrIdempotencyConflict")

	// Test 2: Single approved per intent (uq_tx_one_approved)
	// Mark tx1 as APPROVED
	nowApprove := now.Add(time.Second)
	_ = tx1.TransitionTo(domain.StatusProcessing, nowApprove)
	_ = tx1.MarkApproved("AUT-111", "00", "Approved", nowApprove)
	updated, err := repo.UpdateTransactionStatus(ctx, nil, tx1, domain.StatusPending)
	require.NoError(t, err)
	assert.True(t, updated)

	// Attempt to insert another transaction as APPROVED for the same intent
	tx2 := domain.NewTransaction(uuid.New(), intentID, "m_1001", "idem-new-1", "hash-3", "424242", "4242", corrID, now)
	tx2.Status = domain.StatusApproved // Force status to test physical DB constraint
	err = repo.CreateTransaction(ctx, nil, tx2)
	assert.ErrorIs(t, err, domain.ErrIntentAlreadyApproved, "must map uq_tx_one_approved to ErrIntentAlreadyApproved")

	// Test 3: Single active per intent (uq_tx_one_active)
	// Create another intent
	intentID2 := uuid.New()
	intent2, _ := domain.NewPaymentIntent(intentID2, "m_1001", "ord-active-2", amount, domain.CurrencyCLP, nil, corrID, now)
	_ = repo.CreatePaymentIntent(ctx, nil, intent2)

	txActive1 := domain.NewTransaction(uuid.New(), intentID2, "m_1001", "idem-act-1", "hash-4", "424242", "4242", corrID, now)
	txActive1.Status = domain.StatusProcessing
	err = repo.CreateTransaction(ctx, nil, txActive1)
	require.NoError(t, err)

	// Try to insert a second in-flight active transaction for intent2
	txActive2 := domain.NewTransaction(uuid.New(), intentID2, "m_1001", "idem-act-2", "hash-5", "424242", "4242", corrID, now)
	txActive2.Status = domain.StatusProcessing
	err = repo.CreateTransaction(ctx, nil, txActive2)
	assert.ErrorIs(t, err, domain.ErrActiveTransactionExists, "must map uq_tx_one_active to ErrActiveTransactionExists")
}

func TestRepository_OptimisticLocking(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now().UTC()
	intentID := uuid.New()
	txID := uuid.New()
	orderRef := "ord-lock-" + uuid.New().String()[:8]
	idemKey := "idem-lock-" + uuid.New().String()[:8]

	amount, _ := domain.NewAmount(9990)
	intent, _ := domain.NewPaymentIntent(intentID, "m_1001", orderRef, amount, domain.CurrencyCLP, nil, "corr-lock", now)
	_ = repo.CreatePaymentIntent(ctx, nil, intent)

	txRecord := domain.NewTransaction(txID, intentID, "m_1001", idemKey, "hash-lock", "424242", "4242", "corr-lock", now)
	_ = repo.CreateTransaction(ctx, nil, txRecord)

	// Step 1: Legal update from PENDING to PROCESSING
	_ = txRecord.TransitionTo(domain.StatusProcessing, now)
	updated, err := repo.UpdateTransactionStatus(ctx, nil, txRecord, domain.StatusPending)
	require.NoError(t, err)
	assert.True(t, updated, "should successfully update when fromStatus matches")

	// Step 2: Attempt update with outdated fromStatus (e.g. still sending PENDING)
	txRecord.Status = domain.StatusApproved
	updated, err = repo.UpdateTransactionStatus(ctx, nil, txRecord, domain.StatusPending)
	require.NoError(t, err)
	assert.False(t, updated, "should return false when fromStatus does not match current DB status")
}

func TestRepository_GetTransactionByIdempotencyKey(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now().UTC()
	intentID := uuid.New()
	txID := uuid.New()
	orderRef := "ord-idem-read-" + uuid.New().String()[:8]
	idemKey := "key-lookup-" + uuid.New().String()[:8]

	amount, _ := domain.NewAmount(12000)
	intent, _ := domain.NewPaymentIntent(intentID, "m_1001", orderRef, amount, domain.CurrencyCLP, nil, "corr-read", now)
	_ = repo.CreatePaymentIntent(ctx, nil, intent)

	txRecord := domain.NewTransaction(txID, intentID, "m_1001", idemKey, "hash-read-1", "424242", "4242", "corr-read", now)
	_ = repo.CreateTransaction(ctx, nil, txRecord)

	// Look up by key
	found, err := repo.GetTransactionByIdempotencyKey(ctx, "m_1001", idemKey)
	require.NoError(t, err)
	assert.Equal(t, txID, found.ID)
	assert.Equal(t, "m_1001", found.MerchantID)
	assert.Equal(t, idemKey, found.IdempotencyKey)

	// Non-existent key
	notFound, err := repo.GetTransactionByIdempotencyKey(ctx, "m_1001", "missing-key")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestRepository_ListTransactions(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	now := time.Now().UTC()
	merchantID := "m_1001"

	// Create 3 transactions for m_1001
	for i := 0; i < 3; i++ {
		intentID := uuid.New()
		orderRef := "ord-list-" + uuid.New().String()[:8]
		idemKey := "idem-list-" + uuid.New().String()[:8]
		amount, _ := domain.NewAmount(10000 + int64(i*1000))
		intent, _ := domain.NewPaymentIntent(intentID, merchantID, orderRef, amount, domain.CurrencyCLP, nil, "corr-list", now)
		_ = repo.CreatePaymentIntent(ctx, nil, intent)

		tx := domain.NewTransaction(uuid.New(), intentID, merchantID, idemKey, "hash-list", "424242", "4242", "corr-list", now)
		if i == 0 {
			tx.Status = domain.StatusApproved
		}
		_ = repo.CreateTransaction(ctx, nil, tx)
	}

	// Filter by merchantID
	list, total, err := repo.ListTransactions(ctx, listpayments.Filter{
		MerchantID: &merchantID,
		Limit:      10,
		Offset:     0,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 3)
	assert.GreaterOrEqual(t, len(list), 3)

	// Filter by status APPROVED
	approvedStatus := domain.StatusApproved
	listApproved, totalApproved, err := repo.ListTransactions(ctx, listpayments.Filter{
		MerchantID: &merchantID,
		Status:     &approvedStatus,
		Limit:      10,
		Offset:     0,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, totalApproved, 1)
	for _, item := range listApproved {
		assert.Equal(t, domain.StatusApproved, item.Status)
	}
}

func TestComputePayloadHash(t *testing.T) {
	hash1 := postgres.ComputePayloadHash("m_1001", "ord-1", 25990, "CLP", "424242", "4242", 12, 2028)
	hash2 := postgres.ComputePayloadHash("m_1001", "ord-1", 25990, "CLP", "424242", "4242", 12, 2028)
	assert.Equal(t, hash1, hash2, "identical parameters must produce identical hash")
	assert.Equal(t, 64, len(hash1), "SHA-256 hex string must be 64 characters long")

	// Changing amount must produce different hash
	hashDiffAmount := postgres.ComputePayloadHash("m_1001", "ord-1", 99990, "CLP", "424242", "4242", 12, 2028)
	assert.NotEqual(t, hash1, hashDiffAmount)

	// Changing card last4 must produce different hash
	hashDiffCard := postgres.ComputePayloadHash("m_1001", "ord-1", 25990, "CLP", "424242", "9995", 12, 2028)
	assert.NotEqual(t, hash1, hashDiffCard)
}
