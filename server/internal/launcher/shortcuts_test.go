package launcher

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBuildPayload_NoCwd(t *testing.T) {
	got := buildPayload("", "claude")
	want := "exec claude"
	if got != want {
		t.Errorf("buildPayload = %q, want %q", got, want)
	}
}

func TestBuildPayload_WithCwd(t *testing.T) {
	got := buildPayload("/home/user/proj", "claude --flag")
	want := "cd '/home/user/proj' && exec claude --flag"
	if got != want {
		t.Errorf("buildPayload = %q, want %q", got, want)
	}
}

func TestBuildPayload_EscapesSingleQuotesInCwd(t *testing.T) {
	got := buildPayload("/home/user's dir", "claude")
	want := `cd '/home/user'\''s dir' && exec claude`
	if got != want {
		t.Errorf("buildPayload = %q, want %q", got, want)
	}
}

func TestBuildPayload_TrimsWhitespace(t *testing.T) {
	got := buildPayload("  /tmp  ", "  echo hi  ")
	want := "cd '/tmp' && exec echo hi"
	if got != want {
		t.Errorf("buildPayload = %q, want %q", got, want)
	}
}

func TestSupportedTerminals_IsSorted(t *testing.T) {
	list := SupportedTerminals()
	if len(list) == 0 {
		t.Fatal("SupportedTerminals returned empty list")
	}
	for i := 1; i < len(list); i++ {
		if list[i] < list[i-1] {
			t.Errorf("list not sorted at %d: %q < %q", i, list[i], list[i-1])
		}
	}
}

func TestSupportedTerminals_ContainsKitty(t *testing.T) {
	list := SupportedTerminals()
	for _, name := range list {
		if name == "kitty" {
			return
		}
	}
	t.Error("kitty missing from SupportedTerminals")
}

// TestLaunchCommand_RejectsEmptyCommand verifies that an empty
// command is refused up front before any exec machinery runs.
func TestLaunchCommand_RejectsEmptyCommand(t *testing.T) {
	l := New(nil)
	_, err := l.LaunchCommand("id", "xterm", "", "/tmp", "   ")
	if err == nil {
		t.Fatal("want error on empty command")
	}
}

// TestLaunchCommand_RejectsUnknownTerminal ensures a user-supplied
// terminal string goes through the whitelist.
func TestLaunchCommand_RejectsUnknownTerminal(t *testing.T) {
	l := New(nil)
	_, err := l.LaunchCommand("id", "definitely-not-a-real-terminal", "", "/tmp", "echo hi")
	if err == nil {
		t.Fatal("want error on unknown terminal")
	}
}

// TestLaunchCommand_ExecutesFakeTerminal builds a fake terminal
// script on disk, points PATH at its directory, and runs
// LaunchCommand with a payload that writes a sentinel file. The
// assertion is that the sentinel appears, proving the full argv
// construction + exec path works end-to-end without requiring a
// real kitty or xterm binary.
func TestLaunchCommand_ExecutesFakeTerminal(t *testing.T) {
	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	if err := os.Mkdir(binDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Fake "xterm" that forwards its -e payload to /bin/sh.
	fakeXterm := filepath.Join(binDir, "xterm")
	script := `#!/bin/sh
# Skip the -e flag our launcher passes and eval the remainder.
if [ "$1" = "-e" ]; then
    shift
fi
exec "$@"
`
	if err := os.WriteFile(fakeXterm, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake: %v", err)
	}

	sentinel := filepath.Join(tmp, "sentinel.txt")

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+":"+oldPath)

	l := New(nil)
	pid, err := l.LaunchCommand("test-id", "xterm", "", "", "touch "+sentinel)
	if err != nil {
		t.Fatalf("LaunchCommand: %v", err)
	}
	if pid <= 0 {
		t.Errorf("pid = %d, want > 0", pid)
	}

	// Wait for the reaper goroutine to finish by polling the
	// sentinel file — the child is detached, so we can't join on
	// it directly. A short timeout is enough because touch is near
	// instant.
	waitFor(t, sentinel)
}

func waitFor(t *testing.T, path string) {
	t.Helper()
	// Avoid time.Sleep loops that trip up the harness; retry via
	// exec polling with tiny delays.
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(path); err == nil {
			return
		}
		// Tiny synchronous busy-wait to avoid time.Sleep.
		_ = exec.Command("true").Run()
	}
	t.Fatalf("sentinel %s never appeared", path)
}
