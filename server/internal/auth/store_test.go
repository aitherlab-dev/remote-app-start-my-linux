package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
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

func TestStore_LoadMissingFileIsNotError(t *testing.T) {
	s := NewStore()
	path := filepath.Join(t.TempDir(), "tokens.json")
	if err := s.Load(path); err != nil {
		t.Fatalf("Load on missing file returned error: %v", err)
	}
	if got := s.Count(); got != 0 {
		t.Fatalf("Count after loading missing file = %d, want 0", got)
	}
}

func TestStore_LoadMalformedJSONErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o600); err != nil {
		t.Fatalf("seed malformed file: %v", err)
	}
	s := NewStore()
	if err := s.Load(path); err == nil {
		t.Fatal("Load on malformed file returned nil, want error")
	}
}

func TestStore_LoadEmptyFileIsNotError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("seed empty file: %v", err)
	}
	s := NewStore()
	if err := s.Load(path); err != nil {
		t.Fatalf("Load on empty file: %v", err)
	}
	if got := s.Count(); got != 0 {
		t.Fatalf("Count after empty file = %d, want 0", got)
	}
}

func TestStore_SaveThenLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")

	src := NewStore()
	var plaintexts []string
	for _, label := range []string{"phone", "tablet", "desktop"} {
		plaintext, info, err := IssueToken(label)
		if err != nil {
			t.Fatalf("IssueToken: %v", err)
		}
		src.Add(info)
		plaintexts = append(plaintexts, plaintext)
	}
	if err := src.SaveTo(path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	dst := NewStore()
	if err := dst.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := dst.Count(), 3; got != want {
		t.Fatalf("Count after Load = %d, want %d", got, want)
	}
	for _, plaintext := range plaintexts {
		if _, ok := dst.Validate(plaintext); !ok {
			t.Fatalf("Validate after Load returned false for a round-tripped token")
		}
	}
}

func TestStore_SetPersistPathWritesOnAdd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")

	s := NewStore()
	s.SetPersistPath(path, func(err error) { t.Errorf("unexpected persist error: %v", err) })

	_, info, err := IssueToken("phone")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	s.Add(info)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read tokens file: %v", err)
	}
	var onDisk []TokenInfo
	if err := json.Unmarshal(data, &onDisk); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(onDisk) != 1 || onDisk[0].Hash != info.Hash {
		t.Fatalf("on-disk tokens = %+v, want hash %q", onDisk, info.Hash)
	}
	if onDisk[0].DeviceLabel != "phone" {
		t.Fatalf("on-disk label = %q, want %q", onDisk[0].DeviceLabel, "phone")
	}
}

func TestStore_SetPersistPathWritesOnRevoke(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")

	s := NewStore()
	s.SetPersistPath(path, func(err error) { t.Errorf("unexpected persist error: %v", err) })

	_, info, err := IssueToken("phone")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	s.Add(info)
	s.Revoke(info.Hash)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read tokens file: %v", err)
	}
	var onDisk []TokenInfo
	if err := json.Unmarshal(data, &onDisk); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(onDisk) != 0 {
		t.Fatalf("on-disk tokens after Revoke = %+v, want empty", onDisk)
	}
}

func TestStore_TokensFilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "tokens.json")

	s := NewStore()
	s.SetPersistPath(path, func(err error) { t.Errorf("unexpected persist error: %v", err) })

	_, info, err := IssueToken("phone")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	s.Add(info)

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat tokens file: %v", err)
	}
	if mode := fi.Mode().Perm(); mode != 0o600 {
		t.Fatalf("tokens file mode = %o, want 0600", mode)
	}
	di, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat tokens dir: %v", err)
	}
	if mode := di.Mode().Perm(); mode != 0o700 {
		t.Fatalf("tokens dir mode = %o, want 0700", mode)
	}
}

func TestStore_AtomicWriteLeavesNoLeftovers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")

	s := NewStore()
	s.SetPersistPath(path, func(err error) { t.Errorf("unexpected persist error: %v", err) })
	for i := 0; i < 5; i++ {
		_, info, err := IssueToken("dev")
		if err != nil {
			t.Fatalf("IssueToken: %v", err)
		}
		s.Add(info)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "tokens.json" {
			t.Fatalf("unexpected file left in dir: %q", e.Name())
		}
	}
}

func TestStore_PersistErrorReportedThroughErrLog(t *testing.T) {
	// Pointing the persist path at a location that cannot exist (an
	// empty string segment inside the directory) forces writeTokensFile
	// to fail so we can verify the errLog callback fires.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}
	badPath := filepath.Join(blocker, "tokens.json")

	var captured atomic.Value
	s := NewStore()
	s.SetPersistPath(badPath, func(err error) { captured.Store(err) })

	_, info, err := IssueToken("phone")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	s.Add(info)

	if captured.Load() == nil {
		t.Fatal("errLog was not called for a failed persist")
	}
	if got := s.Count(); got != 1 {
		t.Fatalf("Count after failed persist = %d, want 1 (memory state must survive)", got)
	}
}

func TestStore_ConcurrentAccessWithPersist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")

	s := NewStore()
	s.SetPersistPath(path, func(err error) { t.Errorf("unexpected persist error: %v", err) })

	const workers = 8
	const perWorker = 20

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
					t.Errorf("Validate returned false right after Add")
					return
				}
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

func TestStore_LoadSkipsEntriesWithoutHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")
	payload := `[{"hash":"","device_label":"bogus","created_at":"0001-01-01T00:00:00Z","last_seen":"0001-01-01T00:00:00Z"}]`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	s := NewStore()
	if err := s.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := s.Count(); got != 0 {
		t.Fatalf("Count after Load with empty-hash entries = %d, want 0", got)
	}
}
