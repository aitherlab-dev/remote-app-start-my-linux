package launcher

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sasha/remotelauncher/internal/desktop"
)

// killAfter registers a t.Cleanup that sends SIGTERM to pid if it is
// still alive. Prevents test-leaked background processes.
func killAfter(t *testing.T, pid int) {
	t.Helper()
	t.Cleanup(func() {
		if pid > 0 {
			_ = syscall.Kill(pid, syscall.SIGTERM)
		}
	})
}

// waitForFile polls for path until it exists or timeout elapses.
func waitForFile(t *testing.T, path string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func TestLauncher_EmptyExec(t *testing.T) {
	l := New(nil)
	pid, err := l.Launch(desktop.Entry{Exec: ""})
	if pid != 0 {
		t.Errorf("pid = %d, want 0", pid)
	}
	if !errors.Is(err, ErrEmptyExec) {
		t.Errorf("err = %v, want ErrEmptyExec", err)
	}
}

func TestLauncher_OnlyFieldCodes(t *testing.T) {
	// Exec has only field codes that expand to nothing when files=nil.
	l := New(nil)
	pid, err := l.Launch(desktop.Entry{Exec: "%F"})
	if pid != 0 {
		t.Errorf("pid = %d, want 0", pid)
	}
	if !errors.Is(err, ErrEmptyExec) {
		t.Errorf("err = %v, want ErrEmptyExec", err)
	}
}

func TestLauncher_SplitExecError(t *testing.T) {
	l := New(nil)
	pid, err := l.Launch(desktop.Entry{Exec: `foo "unterminated`})
	if pid != 0 {
		t.Errorf("pid = %d, want 0", pid)
	}
	if err == nil || !strings.Contains(err.Error(), "split exec") {
		t.Errorf("err = %v, want split exec wrap", err)
	}
}

func TestLauncher_BadCommand(t *testing.T) {
	l := New(nil)
	pid, err := l.Launch(desktop.Entry{Exec: "/nonexistent/path/foo"})
	if pid != 0 {
		t.Errorf("pid = %d, want 0", pid)
	}
	if err == nil {
		t.Fatal("err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "start") {
		t.Errorf("err = %v, want wrap containing \"start\"", err)
	}
}

func TestLauncher_StartSucceeds(t *testing.T) {
	tmp := t.TempDir()
	pidFile := filepath.Join(tmp, "pid")
	entry := desktop.Entry{
		ID:   "test-start",
		Exec: `/bin/sh -c "echo $$ > ` + pidFile + `"`,
	}
	l := New(nil)
	pid, err := l.Launch(entry)
	killAfter(t, pid)
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if pid <= 0 {
		t.Fatalf("pid = %d, want >0", pid)
	}
	if !waitForFile(t, pidFile, 2*time.Second) {
		t.Fatalf("pid file %s not created", pidFile)
	}
	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("read pid file: %v", err)
	}
	got, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("parse pid file %q: %v", data, err)
	}
	if got <= 0 {
		t.Errorf("pid in file = %d, want >0", got)
	}
}

func TestLauncher_RunsDetached(t *testing.T) {
	tmp := t.TempDir()
	doneFile := filepath.Join(tmp, "done")
	entry := desktop.Entry{
		ID:   "test-detached",
		Exec: `/bin/sh -c "sleep 1 && touch ` + doneFile + `"`,
	}
	l := New(nil)
	pid, err := l.Launch(entry)
	killAfter(t, pid)
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if pid <= 0 {
		t.Fatalf("pid = %d, want >0", pid)
	}
	// Process must be alive right after Launch (syscall.Kill with
	// signal 0 probes existence: nil or EPERM means alive).
	if err := syscall.Kill(pid, 0); err != nil && !errors.Is(err, syscall.EPERM) {
		t.Errorf("process %d not alive: %v", pid, err)
	}
	if !waitForFile(t, doneFile, 3*time.Second) {
		t.Fatalf("done file %s not created — child did not run independently", doneFile)
	}
}

func TestLauncher_RespectsPath(t *testing.T) {
	tmp := t.TempDir()
	// Resolve symlinks because TempDir on some systems lives under
	// /tmp → /private/tmp and pwd may print the canonical form.
	realTmp, err := filepath.EvalSymlinks(tmp)
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	entry := desktop.Entry{
		ID:   "test-path",
		Exec: `/bin/sh -c "pwd > out"`,
		Path: realTmp,
	}
	l := New(nil)
	pid, err := l.Launch(entry)
	killAfter(t, pid)
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	outFile := filepath.Join(realTmp, "out")
	if !waitForFile(t, outFile, 2*time.Second) {
		t.Fatalf("out file %s not created", outFile)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read out file: %v", err)
	}
	got := strings.TrimSpace(string(data))
	if got != realTmp {
		t.Errorf("pwd = %q, want %q", got, realTmp)
	}
}

func TestLauncher_TerminalTrue_HasEmulator(t *testing.T) {
	if !anyTerminalAvailable() {
		t.Skip("no terminal emulator on PATH")
	}
	entry := desktop.Entry{
		ID:       "test-term",
		Exec:     "true",
		Terminal: true,
	}
	l := New(nil)
	pid, err := l.Launch(entry)
	killAfter(t, pid)
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if pid <= 0 {
		t.Errorf("pid = %d, want >0", pid)
	}
}

func TestLauncher_TerminalTrue_Fallback(t *testing.T) {
	t.Setenv("PATH", "")
	entry := desktop.Entry{
		ID:       "test-term-fallback",
		Exec:     "true",
		Terminal: true,
	}
	l := New(nil)
	pid, err := l.Launch(entry)
	if pid != 0 {
		t.Errorf("pid = %d, want 0", pid)
	}
	if !errors.Is(err, ErrNoTerminalEmulator) {
		t.Errorf("err = %v, want ErrNoTerminalEmulator", err)
	}
}

func anyTerminalAvailable() bool {
	_, err := findTerminal()
	return err == nil
}

func TestLauncher_RegistersPID(t *testing.T) {
	entry := desktop.Entry{
		ID:   "test-register",
		Exec: `/bin/sh -c "sleep 1"`,
	}
	l := New(NewTracker())
	pid, err := l.Launch(entry)
	killAfter(t, pid)
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if pid <= 0 {
		t.Fatalf("pid = %d, want >0", pid)
	}
	if !l.Tracker().Alive(entry.ID) {
		t.Errorf("Alive(%q) = false immediately after Launch, want true", entry.ID)
	}
	gotPids := l.Tracker().Pids(entry.ID)
	if len(gotPids) != 1 || gotPids[0] != pid {
		t.Errorf("Pids(%q) = %v, want [%d]", entry.ID, gotPids, pid)
	}

	// The launched child is still our direct child (Setsid creates a new
	// session but not a new parent), so after sleep exits we must reap
	// it here or kill(pid,0) keeps returning nil on the zombie.
	time.Sleep(1500 * time.Millisecond)
	var ws syscall.WaitStatus
	if _, err := syscall.Wait4(pid, &ws, 0, nil); err != nil {
		t.Fatalf("Wait4(%d): %v", pid, err)
	}

	l.Tracker().Cleanup()
	if l.Tracker().Alive(entry.ID) {
		t.Errorf("Alive(%q) = true after process exit, want false", entry.ID)
	}
}
