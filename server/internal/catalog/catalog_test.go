package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
)

const desktopTemplate = `[Desktop Entry]
Type=Application
Name=%s
Exec=/bin/%s
Comment=%s
Icon=%s
Categories=%s;
`

func writeDesktop(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func writeApp(t *testing.T, dir, file, name, exec, comment, icon, category string) {
	t.Helper()
	content := fmt.Sprintf(desktopTemplate, name, exec, comment, icon, category)
	writeDesktop(t, dir, file, content)
}

func TestCatalog_LoadAndList(t *testing.T) {
	dir := t.TempDir()
	writeApp(t, dir, "banana.desktop", "Banana", "banana", "Yellow fruit", "banana-icon", "Food")
	writeApp(t, dir, "apple.desktop", "Apple", "apple", "Red fruit", "apple-icon", "Food")
	writeApp(t, dir, "cherry.desktop", "Cherry", "cherry", "Tart fruit", "cherry-icon", "Food")

	c := New([]string{dir})
	loaded, scanErrs, err := c.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(scanErrs) != 0 {
		t.Fatalf("unexpected scan errors: %v", scanErrs)
	}
	if loaded != 3 {
		t.Errorf("loaded = %d, want 3", loaded)
	}

	list := c.List()
	if len(list) != 3 {
		t.Fatalf("List len = %d, want 3", len(list))
	}

	wantNames := []string{"Apple", "Banana", "Cherry"}
	for i, want := range wantNames {
		if list[i].Name != want {
			t.Errorf("list[%d].Name = %q, want %q", i, list[i].Name, want)
		}
	}

	// JSON must not leak any execution-related fields.
	out, err := json.Marshal(list)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	lowered := bytes.ToLower(out)
	for _, forbidden := range []string{"exec", "tryexec", "path", "hidden", "onlyshowin", "notshowin", "startupnotify"} {
		if bytes.Contains(lowered, []byte(forbidden)) {
			t.Errorf("JSON output contains forbidden token %q: %s", forbidden, out)
		}
	}
}

func TestCatalog_GetReturnsFullEntry(t *testing.T) {
	dir := t.TempDir()
	writeApp(t, dir, "test.desktop", "Test", "test", "c", "i", "X")

	c := New([]string{dir})
	if _, _, err := c.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	entry, ok := c.Get("test")
	if !ok {
		t.Fatalf("Get(\"test\") = _, false; want true")
	}
	if entry.Exec != "/bin/test" {
		t.Errorf("entry.Exec = %q, want %q", entry.Exec, "/bin/test")
	}
	if entry.Name != "Test" {
		t.Errorf("entry.Name = %q, want Test", entry.Name)
	}
}

func TestCatalog_GetUnknown(t *testing.T) {
	c := New([]string{t.TempDir()})
	if _, _, err := c.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := c.Get("nonexistent"); ok {
		t.Error("Get(\"nonexistent\") = _, true; want false")
	}
}

func TestCatalog_GetInfoUnknown(t *testing.T) {
	c := New([]string{t.TempDir()})
	if _, _, err := c.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := c.GetInfo("nonexistent"); ok {
		t.Error("GetInfo(\"nonexistent\") = _, true; want false")
	}
}

func TestCatalog_ReloadReflectsAdditions(t *testing.T) {
	dir := t.TempDir()
	writeApp(t, dir, "first.desktop", "First", "first", "", "", "")

	c := New([]string{dir})
	if _, _, err := c.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := len(c.List()); got != 1 {
		t.Fatalf("initial List len = %d, want 1", got)
	}

	writeApp(t, dir, "second.desktop", "Second", "second", "", "", "")
	if _, _, err := c.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	if got := len(c.List()); got != 2 {
		t.Errorf("after-reload List len = %d, want 2", got)
	}
}

func TestCatalog_ReloadReflectsRemovals(t *testing.T) {
	dir := t.TempDir()
	writeApp(t, dir, "keep.desktop", "Keep", "keep", "", "", "")
	writeApp(t, dir, "drop.desktop", "Drop", "drop", "", "", "")

	c := New([]string{dir})
	if _, _, err := c.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := len(c.List()); got != 2 {
		t.Fatalf("initial List len = %d, want 2", got)
	}

	if err := os.Remove(filepath.Join(dir, "drop.desktop")); err != nil {
		t.Fatalf("remove drop.desktop: %v", err)
	}
	if _, _, err := c.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	list := c.List()
	if len(list) != 1 {
		t.Fatalf("after-reload List len = %d, want 1", len(list))
	}
	if list[0].ID != "keep" {
		t.Errorf("remaining entry ID = %q, want keep", list[0].ID)
	}
}

func TestCatalog_ReloadReflectsModifications(t *testing.T) {
	dir := t.TempDir()
	writeApp(t, dir, "thing.desktop", "Old", "thing", "", "", "")

	c := New([]string{dir})
	if _, _, err := c.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	info, ok := c.GetInfo("thing")
	if !ok || info.Name != "Old" {
		t.Fatalf("initial GetInfo = %+v, ok=%v; want Name=Old", info, ok)
	}

	writeApp(t, dir, "thing.desktop", "New", "thing", "", "", "")
	if _, _, err := c.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	info, ok = c.GetInfo("thing")
	if !ok {
		t.Fatalf("GetInfo(\"thing\") not found after reload")
	}
	if info.Name != "New" {
		t.Errorf("GetInfo.Name = %q, want New", info.Name)
	}
}

func TestCatalog_ListReturnsCopy(t *testing.T) {
	dir := t.TempDir()
	writeApp(t, dir, "one.desktop", "One", "one", "", "", "")

	c := New([]string{dir})
	if _, _, err := c.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	list := c.List()
	if len(list) == 0 {
		t.Fatal("list is empty")
	}
	list[0].Name = "MUTATED"

	list2 := c.List()
	if list2[0].Name == "MUTATED" {
		t.Error("caller mutation leaked into catalog state")
	}
}

func TestCatalog_ConcurrentReadDuringReload(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		writeApp(t, dir, fmt.Sprintf("app%d.desktop", i), fmt.Sprintf("App%d", i), fmt.Sprintf("a%d", i), "", "", "")
	}

	c := New([]string{dir})
	if _, _, err := c.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				_ = c.List()
				_, _ = c.Get("app0")
				_, _ = c.GetInfo("app1")
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			if _, _, err := c.Reload(); err != nil {
				t.Errorf("Reload: %v", err)
				return
			}
		}
		close(stop)
	}()

	wg.Wait()
}

func TestCatalog_DefaultPathsWhenNil(t *testing.T) {
	c := New(nil)
	loaded, _, err := c.Load()
	if err != nil {
		t.Fatalf("Load with nil paths: %v", err)
	}
	if loaded < 0 {
		t.Errorf("loaded = %d, want ≥0", loaded)
	}
	_ = c.List()
}

func TestCatalog_LoadCollectsScanErrors(t *testing.T) {
	dir := t.TempDir()
	writeApp(t, dir, "good.desktop", "Good", "good", "", "", "")
	writeDesktop(t, dir, "broken.desktop", "not a desktop entry at all\n")

	c := New([]string{dir})
	loaded, scanErrs, err := c.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded != 1 {
		t.Errorf("loaded = %d, want 1 (good should still parse)", loaded)
	}
	if len(scanErrs) != 1 {
		t.Fatalf("scanErrs len = %d, want 1: %v", len(scanErrs), scanErrs)
	}
	if filepath.Base(scanErrs[0].Path) != "broken.desktop" {
		t.Errorf("scan error path = %q, want broken.desktop", scanErrs[0].Path)
	}
}

func TestCatalog_AppInfoSortedByNameThenID(t *testing.T) {
	dir := t.TempDir()
	// Same Name "Twin", different IDs — tie-breaker must be ID.
	writeApp(t, dir, "twin-b.desktop", "Twin", "tb", "", "", "")
	writeApp(t, dir, "twin-a.desktop", "Twin", "ta", "", "", "")
	// Single different entry that sorts before Twin alphabetically.
	writeApp(t, dir, "alpha.desktop", "alpha", "al", "", "", "")

	c := New([]string{dir})
	if _, _, err := c.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	list := c.List()
	if len(list) != 3 {
		t.Fatalf("List len = %d, want 3", len(list))
	}

	// Expect order: alpha, twin-a, twin-b.
	wantIDs := []string{"alpha", "twin-a", "twin-b"}
	gotIDs := []string{list[0].ID, list[1].ID, list[2].ID}
	if !sort.StringsAreSorted(wantIDs) {
		t.Fatal("test setup bug: wantIDs not sorted")
	}
	for i, want := range wantIDs {
		if gotIDs[i] != want {
			t.Errorf("list[%d].ID = %q, want %q (full: %v)", i, gotIDs[i], want, gotIDs)
		}
	}
}
