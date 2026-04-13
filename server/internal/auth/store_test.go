package auth

import (
	"sync"
	"testing"
	"time"
)

func TestStore_AddAndValidate(t *testing.T) {
	s := NewStore()
	plaintext, info, err := IssueToken("phone")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	s.Add(info)

	got, ok := s.Validate(plaintext)
	if !ok {
		t.Fatal("Validate returned ok=false for a stored token")
	}
	if got.Hash != info.Hash {
		t.Fatalf("Validate returned hash %q, want %q", got.Hash, info.Hash)
	}
	if got.DeviceLabel != "phone" {
		t.Fatalf("Validate returned label %q, want %q", got.DeviceLabel, "phone")
	}
}

func TestStore_ValidateUpdatesLastSeen(t *testing.T) {
	s := NewStore()
	plaintext, info, err := IssueToken("phone")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	// Anchor the initial LastSeen in the past so the post-Validate
	// value is unambiguously newer even on clocks with coarse
	// resolution.
	info.LastSeen = time.Now().UTC().Add(-time.Hour)
	s.Add(info)

	got, ok := s.Validate(plaintext)
	if !ok {
		t.Fatal("Validate returned ok=false for a stored token")
	}
	if !got.LastSeen.After(info.LastSeen) {
		t.Fatalf("LastSeen=%v not after initial %v", got.LastSeen, info.LastSeen)
	}
}

func TestStore_ValidateUnknown(t *testing.T) {
	s := NewStore()
	if _, ok := s.Validate("nope"); ok {
		t.Fatal("Validate returned ok=true for an unknown token")
	}
}

func TestStore_Revoke(t *testing.T) {
	s := NewStore()
	plaintext, info, err := IssueToken("phone")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	s.Add(info)
	s.Revoke(info.Hash)
	if _, ok := s.Validate(plaintext); ok {
		t.Fatal("Validate returned ok=true after Revoke")
	}
	// Double revoke must be a no-op.
	s.Revoke(info.Hash)
}

func TestStore_Count(t *testing.T) {
	s := NewStore()
	if got := s.Count(); got != 0 {
		t.Fatalf("empty Count=%d, want 0", got)
	}
	for i := 0; i < 3; i++ {
		_, info, err := IssueToken("dev")
		if err != nil {
			t.Fatalf("IssueToken: %v", err)
		}
		s.Add(info)
	}
	if got := s.Count(); got != 3 {
		t.Fatalf("Count=%d, want 3", got)
	}
}

func TestStore_SnapshotReturnsCopy(t *testing.T) {
	s := NewStore()
	_, info, err := IssueToken("phone")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	s.Add(info)

	snap := s.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("snapshot len=%d, want 1", len(snap))
	}
	snap[0].DeviceLabel = "MUTATED"

	again := s.Snapshot()
	if again[0].DeviceLabel != "phone" {
		t.Fatalf("Store was mutated through snapshot: label=%q", again[0].DeviceLabel)
	}
}

func TestStore_AddDuplicateNoop(t *testing.T) {
	s := NewStore()
	_, info, err := IssueToken("phone")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	s.Add(info)
	mutated := info
	mutated.DeviceLabel = "SHOULD NOT REPLACE"
	s.Add(mutated)

	snap := s.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("snapshot len=%d, want 1", len(snap))
	}
	if snap[0].DeviceLabel != "phone" {
		t.Fatalf("duplicate Add overwrote label: %q", snap[0].DeviceLabel)
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	s := NewStore()
	const workers = 16
	const perWorker = 50

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				plaintext, info, err := IssueToken("dev")
				if err != nil {
					t.Errorf("IssueToken: %v", err)
					return
				}
				s.Add(info)
				if _, ok := s.Validate(plaintext); !ok {
					t.Errorf("Validate returned ok=false right after Add")
					return
				}
				_ = s.Count()
				_ = s.Snapshot()
				s.Revoke(info.Hash)
			}
		}()
	}
	wg.Wait()
	if got := s.Count(); got != 0 {
		t.Fatalf("final Count=%d, want 0", got)
	}
}
