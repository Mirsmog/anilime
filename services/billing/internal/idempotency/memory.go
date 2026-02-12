package idempotency

import (
	"context"
	"sync"
)

// memoryStore is a development-only in-memory idempotency store.
// WARNING: not suitable for production — state is lost on restart and
// does not work across multiple instances.
type memoryStore struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func newMemoryStore() *memoryStore {
	return &memoryStore{seen: make(map[string]struct{})}
}

func (s *memoryStore) Check(_ context.Context, eventID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.seen[eventID]; ok {
		return true, nil
	}
	s.seen[eventID] = struct{}{}
	return false, nil
}
