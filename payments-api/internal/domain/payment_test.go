package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

func TestTransactionStatus_StateMachine(t *testing.T) {
	// Terminal status check
	assert.False(t, domain.StatusPending.IsTerminal())
	assert.False(t, domain.StatusProcessing.IsTerminal())
	assert.True(t, domain.StatusApproved.IsTerminal())
	assert.True(t, domain.StatusDeclined.IsTerminal())
	assert.True(t, domain.StatusFailed.IsTerminal())

	// Legal transitions
	legalTransitions := []struct {
		from domain.TransactionStatus
		to   domain.TransactionStatus
	}{
		{domain.StatusPending, domain.StatusProcessing},
		{domain.StatusPending, domain.StatusDeclined},
		{domain.StatusPending, domain.StatusFailed},
		{domain.StatusProcessing, domain.StatusApproved},
		{domain.StatusProcessing, domain.StatusDeclined},
		{domain.StatusProcessing, domain.StatusFailed},
	}

	for _, tc := range legalTransitions {
		assert.True(t, domain.CanTransition(tc.from, tc.to), "Transition from %s to %s should be allowed", tc.from, tc.to)
	}

	// Illegal transitions from non-terminal states
	illegalTransitions := []struct {
		from domain.TransactionStatus
		to   domain.TransactionStatus
	}{
		{domain.StatusPending, domain.StatusApproved}, // Cannot jump to APPROVED without PROCESSING
		{domain.StatusPending, domain.StatusPending},
		{domain.StatusProcessing, domain.StatusPending},
		{domain.StatusProcessing, domain.StatusProcessing},
	}

	for _, tc := range illegalTransitions {
		assert.False(t, domain.CanTransition(tc.from, tc.to), "Transition from %s to %s should be rejected", tc.from, tc.to)
	}

	// Terminal states cannot transition to ANYTHING
	terminalStates := []domain.TransactionStatus{
		domain.StatusApproved,
		domain.StatusDeclined,
		domain.StatusFailed,
	}

	allStates := []domain.TransactionStatus{
		domain.StatusPending,
		domain.StatusProcessing,
		domain.StatusApproved,
		domain.StatusDeclined,
		domain.StatusFailed,
	}

	for _, term := range terminalStates {
		for _, target := range allStates {
			assert.False(t, domain.CanTransition(term, target), "Terminal state %s cannot transition to %s", term, target)
		}
	}
}

func TestTransaction_MarkApproved(t *testing.T) {
	now := time.Now().UTC()
	tx := domain.NewTransaction(
		uuid.New(),
		uuid.New(),
		"1001",
		"idem-123",
		"hash-sha256",
		"424242",
		"4242",
		"corr-123",
		now,
	)

	// Step 1: Transition to PROCESSING
	require.NoError(t, tx.TransitionTo(domain.StatusProcessing, now))
	assert.Equal(t, domain.StatusProcessing, tx.Status)

	// Step 2: Mark APPROVED
	err := tx.MarkApproved("AUT-123456", "00", "Approved", now.Add(time.Second))
	require.NoError(t, err)
	assert.Equal(t, domain.StatusApproved, tx.Status)
	assert.Equal(t, "AUT-123456", tx.AuthorizationCode)
	assert.Equal(t, "00", tx.ResultCode)
	assert.Equal(t, "Approved", tx.ResultMessage)
	assert.NotNil(t, tx.CompletedAt)

	// Step 3: Attempting any transition on APPROVED must return ErrTerminalStateTransition
	err = tx.TransitionTo(domain.StatusProcessing, now.Add(2*time.Second))
	assert.ErrorIs(t, err, domain.ErrTerminalStateTransition)

	err = tx.MarkDeclined(domain.DeclineSourceAcquirer, "51", "Funds", now.Add(2*time.Second))
	assert.ErrorIs(t, err, domain.ErrTerminalStateTransition)
}

func TestTransaction_MarkDeclined(t *testing.T) {
	now := time.Now().UTC()
	tx := domain.NewTransaction(
		uuid.New(),
		uuid.New(),
		"1001",
		"idem-123",
		"hash-sha256",
		"400000",
		"9995",
		"corr-123",
		now,
	)

	// Can decline directly from PENDING (e.g., merchant inactive or internal rule)
	err := tx.MarkDeclined(domain.DeclineSourceInternal, "merchant_inactive", "Merchant is inactive", now)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusDeclined, tx.Status)
	assert.Equal(t, domain.DeclineSourceInternal, tx.DeclineSource)
	assert.Equal(t, "merchant_inactive", tx.ResultCode)
	assert.Equal(t, "Merchant is inactive", tx.ResultMessage)
	assert.NotNil(t, tx.CompletedAt)

	// Terminal check
	err = tx.MarkApproved("AUT-X", "00", "Approved", now)
	assert.ErrorIs(t, err, domain.ErrTerminalStateTransition)
}

func TestTransaction_MarkFailed(t *testing.T) {
	now := time.Now().UTC()
	tx := domain.NewTransaction(
		uuid.New(),
		uuid.New(),
		"1001",
		"idem-123",
		"hash-sha256",
		"400000",
		"0119",
		"corr-123",
		now,
	)

	require.NoError(t, tx.TransitionTo(domain.StatusProcessing, now))

	err := tx.MarkFailed("acquirer_unavailable", "Acquirer down after retries", now)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusFailed, tx.Status)
	assert.Equal(t, "acquirer_unavailable", tx.ResultCode)
	assert.Equal(t, "Acquirer down after retries", tx.ResultMessage)
	assert.NotNil(t, tx.CompletedAt)

	// Terminal check
	err = tx.TransitionTo(domain.StatusProcessing, now)
	assert.ErrorIs(t, err, domain.ErrTerminalStateTransition)
}

func TestPaymentIntent_Creation(t *testing.T) {
	amt, err := domain.NewAmount(25990)
	require.NoError(t, err)

	now := time.Now().UTC()
	intent, err := domain.NewPaymentIntent(
		uuid.New(),
		"1001",
		"order-999",
		amt,
		domain.CurrencyCLP,
		map[string]any{"customer": "juan@example.com"},
		"corr-intent-1",
		now,
	)
	require.NoError(t, err)
	assert.Equal(t, "1001", intent.MerchantID)
	assert.Equal(t, "order-999", intent.OrderReference)
	assert.Equal(t, int64(25990), intent.Amount.Int64())
	assert.Equal(t, domain.CurrencyCLP, intent.Currency)
}
