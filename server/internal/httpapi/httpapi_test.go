package httpapi

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sasha/remotelauncher/internal/catalog"
)

const desktopTemplate = `[Desktop Entry]
Type=Application
Name=%s
Exec=/bin/%s
Comment=%s
Icon=%s
Categories=%s;
`

// newTestCatalog builds an isolated Catalog rooted in t.TempDir. files
// maps a file name (relative to the temp dir) to its raw contents so
// tests can craft exactly the .desktop corpus they need.
func newTestCatalog(t *testing.T, files map[string]string) *catalog.Catalog {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %q: %v", path, err)
		}
	}
	c := catalog.New([]string{dir})
	if _, _, err := c.Load(); err != nil {
		t.Fatalf("catalog load: %v", err)
	}
	return c
}

// desktopEntry is a convenience wrapper around desktopTemplate.
func desktopEntry(name, exec, comment, icon, category string) string {
	return fmt.Sprintf(desktopTemplate, name, exec, comment, icon, category)
}
