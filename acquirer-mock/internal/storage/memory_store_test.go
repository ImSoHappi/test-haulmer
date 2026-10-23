package storage_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ImSoHappi/test-haulmer/acquirer-mock/internal/domain"
	"github.com/ImSoHappi/test-haulmer/acquirer-mock/internal/storage"
)

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store := storage.NewMemoryStore()
	var wg sync.WaitGroup

	numGoroutines := 50
	wg.Add(numGoroutines * 2)

	for i := 0; i < numGoroutines; i++ {
		refID := fmt.Sprintf("ref-concurrent-%d", i)

		// Goroutine 1: increment attempts
		go func(ref string) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				store.IncrementAttempts(ref)
			}
		}(refID)

		// Goroutine 2: save and retrieve authorizations
		go func(ref string) {
			defer wg.Done()
			store.SaveAuthorization(domain.AuthorizationResponse{
				ReferenceID:       ref,
				Status:            "APPROVED",
				Code:              "00",
				AuthorizationCode: "AUT-CONCURRENT",
				Amount:            1000,
				Currency:          "CLP",
				CreatedAt:         time.Now().UTC(),
			})
			auth, exists := store.GetAuthorization(ref)
			if exists {
				assert.Equal(t, ref, auth.ReferenceID)
			}
		}(refID)
	}

	wg.Wait()

	// Verify all counts
	for i := 0; i < numGoroutines; i++ {
		refID := fmt.Sprintf("ref-concurrent-%d", i)
		assert.Equal(t, 5, store.GetAttempts(refID))
		auth, exists := store.GetAuthorization(refID)
		assert.True(t, exists)
		assert.Equal(t, refID, auth.ReferenceID)
	}

	// Test Reset
	store.Reset()
	assert.Equal(t, 0, store.GetAttempts("ref-concurrent-0"))
	_, exists := store.GetAuthorization("ref-concurrent-0")
	assert.False(t, exists)
}
