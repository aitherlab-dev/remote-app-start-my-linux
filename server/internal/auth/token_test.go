package auth

import (
	"regexp"
	"testing"
)

var base64URLRegexp = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func TestIssueToken_NonEmpty(t *testing.T) {
	plaintext, _, err := IssueToken("phone")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if plaintext == "" {
		t.Fatal("plaintext is empty")
	}
}

func TestIssueToken_Base64URL(t *testing.T) {
	plaintext, _, err := IssueToken("phone")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if !base64URLRegexp.MatchString(plaintext) {
		t.Fatalf("plaintext %q does not match base64url (no padding)", plaintext)
	}
}

func TestIssueToken_HashMatches(t *testing.T) {
	plaintext, info, err := IssueToken("phone")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if got := HashToken(plaintext); got != info.Hash {
		t.Fatalf("HashToken=%q, info.Hash=%q", got, info.Hash)
	}
}

func TestIssueToken_LabelStored(t *testing.T) {
	_, info, err := IssueToken("my pixel")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if info.DeviceLabel != "my pixel" {
		t.Fatalf("DeviceLabel=%q, want %q", info.DeviceLabel, "my pixel")
	}
	if info.CreatedAt.IsZero() {
		t.Fatal("CreatedAt is zero")
	}
	if info.LastSeen.IsZero() {
		t.Fatal("LastSeen is zero")
	}
}

func TestIssueToken_Unique(t *testing.T) {
	a, infoA, err := IssueToken("one")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	b, infoB, err := IssueToken("two")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if a == b {
		t.Fatalf("two tokens collided: %q", a)
	}
	if infoA.Hash == infoB.Hash {
		t.Fatalf("two hashes collided: %q", infoA.Hash)
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	const plaintext = "hello-world"
	first := HashToken(plaintext)
	second := HashToken(plaintext)
	if first != second {
		t.Fatalf("HashToken not deterministic: %q vs %q", first, second)
	}
}
