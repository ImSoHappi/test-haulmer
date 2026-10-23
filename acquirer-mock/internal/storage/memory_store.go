package storage

import (
	"sync"

	"github.com/ImSoHappi/test-haulmer/acquirer-mock/internal/domain"
)

// MemoryStore provides thread-safe in-memory tracking of authorization attempts and outcomes.
type MemoryStore struct {
	mu             sync.RWMutex
	attempts       map[string]int
	authorizations map[string]domain.AuthorizationResponse
}

// NewMemoryStore initializes a new empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		attempts:       make(map[string]int),
		authorizations: make(map[string]domain.AuthorizationResponse),
	}
}

// IncrementAttempts increments and returns the attempt count for a reference ID.
func (s *MemoryStore) IncrementAttempts(referenceID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.attempts[referenceID]++
	return s.attempts[referenceID]
}

// GetAttempts returns the current attempt count for a reference ID.
func (s *MemoryStore) GetAttempts(referenceID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.attempts[referenceID]
}

// SaveAuthorization stores an authorization response indexed by reference ID.
func (s *MemoryStore) SaveAuthorization(resp domain.AuthorizationResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.authorizations[resp.ReferenceID] = resp
}

// GetAuthorization retrieves a previously stored authorization response by reference ID.
func (s *MemoryStore) GetAuthorization(referenceID string) (domain.AuthorizationResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resp, exists := s.authorizations[referenceID]
	return resp, exists
}

// Reset clears all in-memory data (primarily for test isolation).
func (s *MemoryStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.attempts = make(map[string]int)
	s.authorizations = make(map[string]domain.AuthorizationResponse)
}
