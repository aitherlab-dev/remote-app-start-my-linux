//go:build manual
// +build manual

package icons_test

import (
	"testing"

	"github.com/sasha/remotelauncher/internal/icons"
)

// TestFinder_RealSystem is a diagnostic probe of the real user's icon
// themes. It is guarded by the "manual" build tag so it never runs under
// the default test target. Invoke via `make icons-debug`.
//
// The test never fails: even if nothing resolves, the output tells the
// operator what the finder sees, which is the point of the probe.
func TestFinder_RealSystem(t *testing.T) {
	f := icons.New(nil, "")

	t.Logf("base dirs:")
	for _, d := range f.BaseDirs {
		t.Logf("  %s", d)
	}
	t.Logf("theme: %s", f.Theme)

	names := []string{"firefox", "chromium", "vlc", "gimp", "kate"}
	for _, name := range names {
		path, format, err := f.Find(name, 64)
		if err != nil {
			t.Logf("%-10s -> (not found)", name)
			continue
		}
		t.Logf("%-10s -> %s [%s]", name, path, format)
	}
}
