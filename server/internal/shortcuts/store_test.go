package shortcuts

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func mk(id, name, cmd string) Shortcut {
	return Shortcut{ID: id, Name: name, Command: cmd}
}

func TestNewStore_Empty(t *testing.T) {
	s := NewStore()
	if s.Count() != 0 {
		t.Errorf("Count = %d, want 0", s.Count())
	}
	if got := s.List(); len(got) != 0 {
		t.Errorf("List = %v, want empty", got)
	}
	if _, ok := s.Get("anything"); ok {
		t.Error("Get on empty store returned ok")
	}
}

func TestLoad_MissingFileIsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shortcuts.json")

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
	path := filepath.Join(dir, "shortcuts.json")
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
	path := filepath.Join(dir, "shortcuts.json")
	content := `{"shortcuts":[
		{"id":"a","name":"Alpha","command":"alpha","cwd":"/tmp","terminal":"kitty","icon":"A"},
		{"id":"b","name":"Beta","command":"beta"}
	]}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	s := NewStore()
	if err := s.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := s.Count(), 2; got != want {
		t.Errorf("Count = %d, want %d", got, want)
	}
	a, ok := s.Get("a")
	if !ok {
		t.Fatal("a not found")
	}
	if a.Name != "Alpha" || a.Cwd != "/tmp" || a.Terminal != "kitty" || a.Icon != "A" {
		t.Errorf("a = %+v", a)
	}
	if _, ok := s.Get("b"); !ok {
		t.Error("b not found")
	}
}

func TestLoad_DuplicateIdErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shortcuts.json")
	content := `{"shortcuts":[{"id":"a","name":"A","command":"c"},{"id":"a","name":"A2","command":"c2"}]}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	s := NewStore()
	if err := s.Load(path); err == nil {
		t.Fatal("Load: expected duplicate id error")
	}
}

func TestLoad_MalformedFileErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shortcuts.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	s := NewStore()
	if err := s.Load(path); err == nil {
		t.Fatal("Load: expected parse error")
	}
}

func TestReplace_PersistsAndReloads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shortcuts.json")

	s := NewStore()
	if err := s.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := s.Replace([]Shortcut{
		mk("claude-a", "Claude A", "claude"),
		mk("claude-b", "Claude B", "claude"),
	}); err != nil {
		t.Fatalf("Replace: %v", err)
	}

	other := NewStore()
	if err := other.Load(path); err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if got, want := other.Count(), 2; got != want {
		t.Errorf("Count = %d, want %d", got, want)
	}
	if sc, ok := other.Get("claude-a"); !ok || sc.Name != "Claude A" {
		t.Errorf("claude-a = %+v, ok=%v", sc, ok)
	}
}

func TestReplace_OverwritesPrevious(t *testing.T) {
	dir := t.TempDir()
	s := NewStore()
	if err := s.Load(filepath.Join(dir, "shortcuts.json")); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := s.Replace([]Shortcut{mk("one", "One", "one")}); err != nil {
		t.Fatalf("Replace 1: %v", err)
	}
	if err := s.Replace([]Shortcut{mk("two", "Two", "two")}); err != nil {
		t.Fatalf("Replace 2: %v", err)
	}
	if _, ok := s.Get("one"); ok {
		t.Error("one survived overwrite")
	}
	if _, ok := s.Get("two"); !ok {
		t.Error("two missing after overwrite")
	}
}

func TestReplace_RejectsBlankFields(t *testing.T) {
	cases := []struct {
		name string
		sc   Shortcut
	}{
		{"blank id", Shortcut{Name: "n", Command: "c"}},
		{"blank name", Shortcut{ID: "i", Command: "c"}},
		{"blank command", Shortcut{ID: "i", Name: "n"}},
		{"id with space", Shortcut{ID: "a b", Name: "n", Command: "c"}},
		{"id with slash", Shortcut{ID: "a/b", Name: "n", Command: "c"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewStore()
			err := s.Replace([]Shortcut{tc.sc})
			if err == nil {
				t.Fatal("want error, got nil")
			}
			if !errors.Is(err, ErrInvalidShortcut) {
				t.Errorf("want ErrInvalidShortcut, got %v", err)
			}
		})
	}
}

func TestReplace_RejectsDuplicateIDs(t *testing.T) {
	s := NewStore()
	err := s.Replace([]Shortcut{
		mk("dup", "A", "a"),
		mk("dup", "B", "b"),
	})
	if err == nil {
		t.Fatal("want error, got nil")
	}
}

func TestReplace_LeavesOldStateOnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shortcuts.json")

	s := NewStore()
	if err := s.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := s.Replace([]Shortcut{mk("good", "Good", "cmd")}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Try to replace with an invalid entry.
	err := s.Replace([]Shortcut{mk("good", "Good", "cmd"), mk("", "", "")})
	if err == nil {
		t.Fatal("want error on invalid, got nil")
	}
	if _, ok := s.Get("good"); !ok {
		t.Error("old state lost after failed replace")
	}
}

func TestReplace_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "remotelauncher", "shortcuts.json")
	s := NewStore()
	if err := s.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := s.Replace([]Shortcut{mk("x", "X", "x")}); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not written: %v", err)
	}
}

func TestNormalizeID(t *testing.T) {
	cases := []struct {
		in     string
		want   string
		prefix bool
	}{
		{"custom:foo", "foo", true},
		{"foo", "foo", false},
		{"custom:", "", true},
		{"custom:a:b", "a:b", true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, p := NormalizeID(tc.in)
			if got != tc.want || p != tc.prefix {
				t.Errorf("NormalizeID(%q) = (%q, %v), want (%q, %v)", tc.in, got, p, tc.want, tc.prefix)
			}
		})
	}
}

func TestPrefixedID(t *testing.T) {
	if got, want := PrefixedID("foo"), "custom:foo"; got != want {
		t.Errorf("PrefixedID = %q, want %q", got, want)
	}
}

func TestList_PreservesInsertionOrder(t *testing.T) {
	s := NewStore()
	if err := s.Replace([]Shortcut{
		mk("zeta", "Z", "z"),
		mk("alpha", "A", "a"),
		mk("mu", "M", "m"),
	}); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	got := s.List()
	if len(got) != 3 {
		t.Fatalf("len = %d", len(got))
	}
	wantOrder := []string{"zeta", "alpha", "mu"}
	for i, sc := range got {
		if sc.ID != wantOrder[i] {
			t.Errorf("position %d: id = %q, want %q", i, sc.ID, wantOrder[i])
		}
	}
}
