package visibility

import (
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
)

func TestNewStore_Empty(t *testing.T) {
	s := NewStore()
	if s.Count() != 0 {
		t.Errorf("Count = %d, want 0", s.Count())
	}
	if s.IsHidden("anything") {
		t.Error("IsHidden = true on empty store")
	}
	if got := s.Hidden(); len(got) != 0 {
		t.Errorf("Hidden = %v, want empty", got)
	}
}

func TestLoad_MissingFileIsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "visibility.json")

	s := NewStore()
	if err := s.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Count() != 0 {
		t.Errorf("Count = %d, want 0", s.Count())
	}
	if s.Path() != path {
		t.Errorf("Path = %q, want %q", s.Path(), path)
	}
}

func TestLoad_EmptyFileIsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "visibility.json")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	s := NewStore()
	if err := s.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Count() != 0 {
		t.Errorf("Count = %d, want 0", s.Count())
	}
}

func TestLoad_PopulatedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "visibility.json")
	content := `{"hidden": ["chromium", "firefox", ""]}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	s := NewStore()
	if err := s.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !s.IsHidden("chromium") {
		t.Error("chromium not hidden")
	}
	if !s.IsHidden("firefox") {
		t.Error("firefox not hidden")
	}
	if s.IsHidden("") {
		t.Error("empty id treated as hidden")
	}
	if got, want := s.Count(), 2; got != want {
		t.Errorf("Count = %d, want %d", got, want)
	}
	got := s.Hidden()
	want := []string{"chromium", "firefox"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("Hidden = %v, want %v", got, want)
	}
}

func TestLoad_MalformedFileErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "visibility.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	s := NewStore()
	if err := s.Load(path); err == nil {
		t.Fatal("Load: want error on malformed JSON, got nil")
	}
}

func TestSetHidden_PersistsToDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "visibility.json")

	s := NewStore()
	if err := s.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err := s.SetHidden([]string{"chromium", "firefox", "chromium", ""}); err != nil {
		t.Fatalf("SetHidden: %v", err)
	}
	if got, want := s.Count(), 2; got != want {
		t.Errorf("Count = %d, want %d", got, want)
	}

	// Round-trip through a fresh Store reading the same file.
	other := NewStore()
	if err := other.Load(path); err != nil {
		t.Fatalf("second Load: %v", err)
	}
	got := other.Hidden()
	want := []string{"chromium", "firefox"}
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("Hidden = %v, want %v", got, want)
	}
}

func TestSetHidden_OverwritesPrevious(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "visibility.json")

	s := NewStore()
	if err := s.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := s.SetHidden([]string{"one", "two"}); err != nil {
		t.Fatalf("SetHidden: %v", err)
	}
	if err := s.SetHidden([]string{"three"}); err != nil {
		t.Fatalf("SetHidden 2: %v", err)
	}

	if s.IsHidden("one") || s.IsHidden("two") {
		t.Error("previous entries survived overwrite")
	}
	if !s.IsHidden("three") {
		t.Error("three not hidden")
	}
}

func TestSetHidden_WithoutPathKeepsInMemory(t *testing.T) {
	s := NewStore()
	if err := s.SetHidden([]string{"a"}); err != nil {
		t.Fatalf("SetHidden: %v", err)
	}
	if !s.IsHidden("a") {
		t.Error("a not hidden")
	}
}

func TestSetHidden_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "remotelauncher", "visibility.json")

	s := NewStore()
	if err := s.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := s.SetHidden([]string{"x"}); err != nil {
		t.Fatalf("SetHidden: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not written: %v", err)
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "visibility.json")

	s := NewStore()
	if err := s.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := s.SetHidden([]string{"chromium"}); err != nil {
		t.Fatalf("SetHidden: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = s.IsHidden("chromium")
			_ = s.Hidden()
			_ = s.Count()
		}()
		go func(i int) {
			defer wg.Done()
			ids := []string{"chromium"}
			if i%2 == 0 {
				ids = append(ids, "firefox")
			}
			_ = s.SetHidden(ids)
		}(i)
	}
	wg.Wait()

	if !s.IsHidden("chromium") {
		t.Error("chromium not hidden after concurrent writes")
	}
}

func TestSetPath_ChangesWriteLocation(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.json")
	second := filepath.Join(dir, "second.json")

	s := NewStore()
	if err := s.Load(first); err != nil {
		t.Fatalf("Load: %v", err)
	}
	s.SetPath(second)
	if err := s.SetHidden([]string{"one"}); err != nil {
		t.Fatalf("SetHidden: %v", err)
	}
	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Errorf("first file should not exist, stat err = %v", err)
	}
	if _, err := os.Stat(second); err != nil {
		t.Errorf("second file missing: %v", err)
	}
}
