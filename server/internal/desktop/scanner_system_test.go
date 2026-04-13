//go:build manual
// +build manual

package desktop_test

import (
	"testing"

	"github.com/sasha/remotelauncher/internal/desktop"
)

func TestScan_RealSystem(t *testing.T) {
	paths := desktop.DefaultPaths()
	entries, errs := desktop.Scan(paths)
	t.Logf("found %d entries, %d scan errors from %d paths", len(entries), len(errs), len(paths))
	for _, e := range errs {
		t.Logf("  scan error: %v", &e)
	}
	if len(entries) == 0 {
		t.Fatalf("expected at least 1 entry on a real Linux system")
	}
}
