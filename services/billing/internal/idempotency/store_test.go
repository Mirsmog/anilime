package idempotency

import (
	"context"
	"testing"
)

func TestMemoryStore_FirstCallIsNotDuplicate(t *testing.T) {
	s := newMemoryStore()
	dup, err := s.Check(context.Background(), "evt_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dup {
		t.Fatal("first check should not be duplicate")
	}
}

func TestMemoryStore_SecondCallIsDuplicate(t *testing.T) {
	s := newMemoryStore()
	ctx := context.Background()

	_, _ = s.Check(ctx, "evt_002")

	dup, err := s.Check(ctx, "evt_002")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dup {
		t.Fatal("second check should be duplicate")
	}
}

func TestMemoryStore_DifferentEventsAreIndependent(t *testing.T) {
	s := newMemoryStore()
	ctx := context.Background()

	_, _ = s.Check(ctx, "evt_A")

	dup, err := s.Check(ctx, "evt_B")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dup {
		t.Fatal("different event IDs should not collide")
	}
}

func TestNewStore_FallsBackToMemory(t *testing.T) {
	s, err := NewStore("", "", 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := s.(*memoryStore); !ok {
		t.Fatalf("expected memoryStore when no DSN provided, got %T", s)
	}
}

func TestNewStore_RejectsMemoryInProd(t *testing.T) {
	s, err := NewStore("", "", 0, true)
	if err == nil {
		t.Fatalf("expected error in production with no DSN, got store %T", s)
	}
	if s != nil {
		t.Fatalf("expected nil store, got %T", s)
	}
}
