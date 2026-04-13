package desktop

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	fakeAppsSystem = "testdata/fake_apps/system/applications"
	fakeAppsUser   = "testdata/fake_apps/user/applications"
)

func indexByID(entries []Entry) map[string]Entry {
	out := make(map[string]Entry, len(entries))
	for _, e := range entries {
		out[e.ID] = e
	}
	return out
}

func TestScan_LoadsValidEntries(t *testing.T) {
	entries, errs := Scan([]string{fakeAppsSystem, fakeAppsUser})

	got := indexByID(entries)
	for _, id := range []string{"valid1", "valid2", "kde-foo", "usonly"} {
		if _, ok := got[id]; !ok {
			t.Errorf("expected entry %q in results, missing", id)
		}
	}

	for _, id := range []string{"nodisplay", "hidden", "link", "broken", "random"} {
		if _, ok := got[id]; ok {
			t.Errorf("entry %q should have been filtered out", id)
		}
	}

	if len(got) != 4 {
		t.Errorf("expected exactly 4 entries, got %d: %v", len(got), entryIDs(entries))
	}

	// broken.desktop must appear in scan errors, nothing else.
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 scan error, got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Path, "broken.desktop") {
		t.Errorf("scan error path = %q, want to contain broken.desktop", errs[0].Path)
	}
}

func TestScan_MergesUserOverSystem(t *testing.T) {
	entries, _ := Scan([]string{fakeAppsSystem, fakeAppsUser})

	got := indexByID(entries)
	e, ok := got["valid1"]
	if !ok {
		t.Fatalf("entry valid1 missing")
	}
	if e.Name != "user-version" {
		t.Errorf("valid1.Name = %q, want %q (user should override system)", e.Name, "user-version")
	}
}

func TestScan_MergesSystemOverUserWhenOrderReversed(t *testing.T) {
	// Scan honours input order: the *last* directory wins.
	entries, _ := Scan([]string{fakeAppsUser, fakeAppsSystem})

	got := indexByID(entries)
	e, ok := got["valid1"]
	if !ok {
		t.Fatalf("entry valid1 missing")
	}
	if e.Name != "system-version" {
		t.Errorf("valid1.Name = %q, want %q (system last → wins)", e.Name, "system-version")
	}
}

func TestScan_GeneratesNestedID(t *testing.T) {
	entries, _ := Scan([]string{fakeAppsSystem})

	got := indexByID(entries)
	e, ok := got["kde-foo"]
	if !ok {
		t.Fatalf("expected ID kde-foo, got ids: %v", entryIDs(entries))
	}
	if e.Name != "KDE Foo" {
		t.Errorf("kde-foo Name = %q, want %q", e.Name, "KDE Foo")
	}
}

func TestScan_CollectsScanErrors(t *testing.T) {
	entries, errs := Scan([]string{fakeAppsSystem})
	if len(errs) != 1 {
		t.Fatalf("expected 1 scan error, got %d: %v", len(errs), errs)
	}
	if filepath.Base(errs[0].Path) != "broken.desktop" {
		t.Errorf("scan error path = %q, want basename broken.desktop", errs[0].Path)
	}
	if errs[0].Unwrap() == nil {
		t.Error("ScanError.Unwrap should return the underlying parse error")
	}
	if msg := (&errs[0]).Error(); !strings.Contains(msg, "broken.desktop") {
		t.Errorf("ScanError.Error() = %q, want to mention broken.desktop", msg)
	}

	// The rest of the walk must have continued.
	if len(entries) == 0 {
		t.Error("scan must still return entries despite broken.desktop")
	}
}

func TestScan_NonExistentDirIsNotError(t *testing.T) {
	entries, errs := Scan([]string{"testdata/does-not-exist", fakeAppsUser})
	if len(errs) != 0 {
		t.Errorf("unexpected scan errors: %v", errs)
	}
	got := indexByID(entries)
	if _, ok := got["usonly"]; !ok {
		t.Error("expected usonly from the existing user dir")
	}
}

func TestScan_EmptyPaths(t *testing.T) {
	entries, errs := Scan(nil)
	if entries != nil || errs != nil {
		t.Errorf("Scan(nil) = (%v, %v), want (nil, nil)", entries, errs)
	}
}

func TestScan_SortedByID(t *testing.T) {
	entries, _ := Scan([]string{fakeAppsSystem, fakeAppsUser})
	ids := entryIDs(entries)
	if !sort.StringsAreSorted(ids) {
		t.Errorf("entries not sorted by ID: %v", ids)
	}
}

func TestScan_SkipsFileAsPath(t *testing.T) {
	// Passing a file (not a directory) as a scan root should be a silent skip.
	entries, errs := Scan([]string{"testdata/fake_apps/system/applications/valid1.desktop"})
	if len(entries) != 0 {
		t.Errorf("expected no entries, got %v", entryIDs(entries))
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestScan_UnreadableDesktopFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root ignores file permissions")
	}

	root := t.TempDir()

	good := filepath.Join(root, "good.desktop")
	if err := os.WriteFile(good, []byte("[Desktop Entry]\nType=Application\nName=Good\nExec=/bin/good\n"), 0o644); err != nil {
		t.Fatalf("write good.desktop: %v", err)
	}

	bad := filepath.Join(root, "bad.desktop")
	if err := os.WriteFile(bad, []byte("[Desktop Entry]\nType=Application\nName=Bad\nExec=/bin/bad\n"), 0o000); err != nil {
		t.Fatalf("write bad.desktop: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(bad, 0o644) })

	entries, errs := Scan([]string{root})

	found := false
	for _, e := range entries {
		if e.ID == "good" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected good.desktop to be scanned, got entries: %v", entryIDs(entries))
	}

	if len(errs) != 1 {
		t.Fatalf("expected 1 scan error for unreadable file, got %d: %v", len(errs), errs)
	}
	if filepath.Base(errs[0].Path) != "bad.desktop" {
		t.Errorf("scan error path = %q, want basename bad.desktop", errs[0].Path)
	}
}

func TestComputeID(t *testing.T) {
	tests := []struct {
		root, path string
		want       string
		wantErr    bool
	}{
		{"/apps", "/apps/firefox.desktop", "firefox", false},
		{"/apps", "/apps/kde/foo.desktop", "kde-foo", false},
		{"/apps", "/apps/a/b/c.desktop", "a-b-c", false},
		// Relative root + absolute path cannot be reconciled without cwd:
		// filepath.Rel returns an error.
		{"relative/root", "/absolute/path.desktop", "", true},
	}
	for _, tc := range tests {
		got, err := computeID(tc.root, tc.path)
		if tc.wantErr {
			if err == nil {
				t.Errorf("computeID(%q, %q): want error, got %q", tc.root, tc.path, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("computeID(%q, %q): unexpected error: %v", tc.root, tc.path, err)
			continue
		}
		if got != tc.want {
			t.Errorf("computeID(%q, %q) = %q, want %q", tc.root, tc.path, got, tc.want)
		}
	}
}

func TestScan_UnreadableSubdirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root ignores directory permissions")
	}

	root := t.TempDir()

	good := filepath.Join(root, "good.desktop")
	if err := os.WriteFile(good, []byte("[Desktop Entry]\nType=Application\nName=Good\nExec=/bin/good\n"), 0o644); err != nil {
		t.Fatalf("write good.desktop: %v", err)
	}

	locked := filepath.Join(root, "locked")
	if err := os.Mkdir(locked, 0o755); err != nil {
		t.Fatalf("mkdir locked: %v", err)
	}
	if err := os.WriteFile(filepath.Join(locked, "inner.desktop"), []byte("[Desktop Entry]\nType=Application\nName=Inner\nExec=/bin/inner\n"), 0o644); err != nil {
		t.Fatalf("write inner.desktop: %v", err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatalf("chmod locked: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	entries, errs := Scan([]string{root})

	found := false
	for _, e := range entries {
		if e.ID == "good" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected good.desktop from root, got: %v", entryIDs(entries))
	}
	if len(errs) == 0 {
		t.Error("expected at least one scan error from the locked directory")
	}
}

func entryIDs(entries []Entry) []string {
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.ID)
	}
	return ids
}

func TestDefaultPaths_NoXDG(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_DATA_DIRS", "")
	t.Setenv("HOME", "/tmp/fakehome")

	got := DefaultPaths()

	want := []string{
		"/usr/share/applications",
		"/usr/local/share/applications",
		"/tmp/fakehome/.local/share/applications",
	}
	if !stringSlicesEqual(got, want) {
		t.Errorf("DefaultPaths() = %v, want %v", got, want)
	}
}

func TestDefaultPaths_WithXDG(t *testing.T) {
	t.Setenv("XDG_DATA_DIRS", "/foo:/bar")
	t.Setenv("XDG_DATA_HOME", "/baz")
	t.Setenv("HOME", "/tmp/irrelevant")

	got := DefaultPaths()

	for _, want := range []string{
		"/foo/applications",
		"/bar/applications",
		"/baz/applications",
	} {
		if !contains(got, want) {
			t.Errorf("DefaultPaths() = %v, missing %q", got, want)
		}
	}
	// XDG_DATA_HOME must be last (highest priority).
	if got[len(got)-1] != "/baz/applications" {
		t.Errorf("last path = %q, want %q (XDG_DATA_HOME wins)", got[len(got)-1], "/baz/applications")
	}
}

func TestDefaultPaths_DeduplicatesPaths(t *testing.T) {
	t.Setenv("XDG_DATA_DIRS", "/usr/share:/usr/local/share")
	t.Setenv("XDG_DATA_HOME", "/home/u/.local/share")
	t.Setenv("HOME", "/home/u")

	got := DefaultPaths()

	seen := make(map[string]int)
	for _, p := range got {
		seen[p]++
	}
	for p, n := range seen {
		if n > 1 {
			t.Errorf("path %q appears %d times, expected 1", p, n)
		}
	}
}

func TestDefaultPaths_IgnoresEmptyXDGEntries(t *testing.T) {
	t.Setenv("XDG_DATA_DIRS", "/foo::/bar:")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/tmp/fakehome")

	got := DefaultPaths()

	if contains(got, "/applications") {
		t.Errorf("DefaultPaths() contains empty-derived entry: %v", got)
	}
	for _, want := range []string{"/foo/applications", "/bar/applications"} {
		if !contains(got, want) {
			t.Errorf("DefaultPaths() = %v, missing %q", got, want)
		}
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
