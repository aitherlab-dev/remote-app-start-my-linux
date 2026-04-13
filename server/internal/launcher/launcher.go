// Package launcher starts desktop applications detached from the
// server process so they survive server restarts.
//
// Launch resolves argv via desktop.SplitExec + desktop.Expand, optionally
// wraps the command in a terminal emulator for Terminal=true entries, and
// starts it with SysProcAttr.Setsid=true. Child stdio is nullified. After
// Start, a reaper goroutine calls cmd.Wait so the child is not left as a
// zombie while the server is running, and then drops the pid from the
// Tracker via Forget so Alive flips to false promptly on exit.
package launcher

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/sasha/remotelauncher/internal/desktop"
)

// Sentinel errors returned by Launcher.Launch.
var (
	ErrEmptyExec          = errors.New("empty exec")
	ErrNoTerminalEmulator = errors.New("no terminal emulator found")
)

// terminalCandidates is the ordered list of terminal emulators tried
// when an entry has Terminal=true. The first one found on PATH wins.
var terminalCandidates = []string{
	"x-terminal-emulator",
	"kitty",
	"alacritty",
	"gnome-terminal",
	"konsole",
	"xterm",
}

// Launcher starts applications detached from the current process.
// It always registers successfully started PIDs with its Tracker so
// callers can later ask whether an app is running.
type Launcher struct {
	tracker *Tracker
}

// New returns a Launcher bound to tracker. If tracker is nil a fresh
// empty Tracker is created internally.
func New(tracker *Tracker) *Launcher {
	if tracker == nil {
		tracker = NewTracker()
	}
	return &Launcher{tracker: tracker}
}

// Tracker returns the Tracker this Launcher registers PIDs with.
func (l *Launcher) Tracker() *Tracker { return l.tracker }

// Launch starts the application described by entry and returns the PID
// of the started process. The child is detached via Setsid, so it keeps
// running after the server exits. Stdin/stdout/stderr are nil. Launch
// returns immediately; a background goroutine reaps the child via
// cmd.Wait and removes its pid from the Tracker on exit.
func (l *Launcher) Launch(entry desktop.Entry) (int, error) {
	if entry.Exec == "" {
		return 0, ErrEmptyExec
	}

	tokens, err := desktop.SplitExec(entry.Exec)
	if err != nil {
		return 0, fmt.Errorf("split exec: %w", err)
	}
	args := desktop.Expand(tokens, &entry, nil)
	if len(args) == 0 {
		return 0, ErrEmptyExec
	}

	if entry.Terminal {
		term, err := findTerminal()
		if err != nil {
			return 0, err
		}
		args = append([]string{term, "-e"}, args...)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if entry.Path != "" {
		cmd.Dir = entry.Path
	}
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start %q: %w", entry.ID, err)
	}
	pid := cmd.Process.Pid
	l.tracker.Register(entry.ID, pid)

	// Reap the child in a goroutine so it does not become a zombie
	// while the server is still running. Setsid detaches the process
	// from our session but not from us as parent — init only takes
	// over after we exit. Forget removes the pid from the tracker so
	// Alive flips to false as soon as the child dies.
	go func(cmd *exec.Cmd, id string, pid int) {
		_ = cmd.Wait()
		l.tracker.Forget(id, pid)
	}(cmd, entry.ID, pid)

	return pid, nil
}

// findTerminal returns the absolute path of the first terminal emulator
// from terminalCandidates available on PATH, or ErrNoTerminalEmulator.
func findTerminal() (string, error) {
	for _, name := range terminalCandidates {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", ErrNoTerminalEmulator
}
