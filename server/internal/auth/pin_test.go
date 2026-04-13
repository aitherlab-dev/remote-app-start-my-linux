package auth

import (
	"strings"
	"testing"
	"time"
)

func TestGeneratePIN_Length(t *testing.T) {
	for i := 0; i < 20; i++ {
		pin, err := GeneratePIN()
		if err != nil {
			t.Fatalf("GeneratePIN: %v", err)
		}
		if len(pin) != 6 {
			t.Fatalf("pin %q has length %d, want 6", pin, len(pin))
		}
	}
}

func TestGeneratePIN_NumericOnly(t *testing.T) {
	const digits = "0123456789"
	for i := 0; i < 20; i++ {
		pin, err := GeneratePIN()
		if err != nil {
			t.Fatalf("GeneratePIN: %v", err)
		}
		for _, r := range pin {
			if !strings.ContainsRune(digits, r) {
				t.Fatalf("pin %q contains non-digit %q", pin, r)
			}
		}
	}
}

func TestPINSession_Generates6Digits(t *testing.T) {
	s, err := NewPINSession(0)
	if err != nil {
		t.Fatalf("NewPINSession: %v", err)
	}
	pin := s.Current()
	if len(pin) != 6 {
		t.Fatalf("pin %q has length %d, want 6", pin, len(pin))
	}
	for _, r := range pin {
		if r < '0' || r > '9' {
			t.Fatalf("pin %q contains non-digit %q", pin, r)
		}
	}
}

func TestPINSession_ConsumeOnce(t *testing.T) {
	s, err := NewPINSession(0)
	if err != nil {
		t.Fatalf("NewPINSession: %v", err)
	}
	if !s.Consume() {
		t.Fatal("first Consume returned false")
	}
	if s.Consume() {
		t.Fatal("second Consume returned true")
	}
}

func TestPINSession_Expires(t *testing.T) {
	s, err := NewPINSession(1 * time.Millisecond)
	if err != nil {
		t.Fatalf("NewPINSession: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	if s.Consume() {
		t.Fatal("Consume returned true after expiration")
	}
}

func TestPINSession_String(t *testing.T) {
	s, err := NewPINSession(0)
	if err != nil {
		t.Fatalf("NewPINSession: %v", err)
	}
	str := s.String()
	if !strings.Contains(str, s.Current()) {
		t.Fatalf("String()=%q does not contain pin %q", str, s.Current())
	}
}

func TestGeneratePIN_Distribution(t *testing.T) {
	const n = 100
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		pin, err := GeneratePIN()
		if err != nil {
			t.Fatalf("GeneratePIN: %v", err)
		}
		seen[pin] = struct{}{}
	}
	// The PIN space has a million entries — a crypto-random generator
	// should never collapse 100 samples into a tiny set. A threshold of
	// ">5 distinct" catches stuck or trivially-seeded generators while
	// keeping the false-positive rate negligible.
	if len(seen) <= 5 {
		t.Fatalf("PIN distribution too narrow: %d unique of %d", len(seen), n)
	}
}
