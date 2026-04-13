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
// Unlike the unit tests, this one is allowed to depend on the host
// system. We don't know which theme is actually active on the developer
// box, so the probe iterates a list of common themes and reports per-
// theme hit counts. The S2.3 success criterion is that at least one of
// the probed themes resolves >= 4 of 5 well-known application icons —
// below that bar the resolver is still missing what S2.3 was supposed
// to fix, and the test fails loudly.
func TestFinder_RealSystem(t *testing.T) {
	names := []string{"firefox", "chromium", "vlc", "gimp", "kate"}
	themes := []string{
		"hicolor",
		"Adwaita",
		"Breeze",
		"breeze",
		"Papirus",
		"Gruvbox Plus Dark",
		"Gruvbox-Plus-Dark",
	}
	const minHits = 4

	base := icons.New(nil, "")
	t.Logf("base dirs:")
	for _, d := range base.BaseDirs {
		t.Logf("  %s", d)
	}

	best := 0
	bestTheme := ""
	for _, theme := range themes {
		f := icons.New(nil, theme)
		var hits int
		for _, name := range names {
			path, format, err := f.Find(name, 64)
			if err != nil {
				t.Logf("[%s] %-10s -> (not found)", theme, name)
				continue
			}
			hits++
			t.Logf("[%s] %-10s -> %s [%s]", theme, name, path, format)
		}
		t.Logf("[%s] resolved %d of %d", theme, hits, len(names))
		if hits > best {
			best = hits
			bestTheme = theme
		}
	}

	t.Logf("best: %d of %d via %q", best, len(names), bestTheme)
	if best < minHits {
		t.Errorf("best theme resolved %d of %d icons, want >= %d", best, len(names), minHits)
	}
}
