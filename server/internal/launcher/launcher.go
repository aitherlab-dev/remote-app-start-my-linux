// Package launcher starts desktop applications detached from the
// server process so they survive server restarts.
//
// Launch resolves argv via desktop.SplitExec + desktop.Expand, optionally
// wraps the command in a terminal emulator for Terminal=true entries, and
// starts it with SysProcAttr.Setsid=true. Child stdio is nullified and
// no Wait is performed — this is fire-and-forget.
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
// It is safe to use a zero value or one returned by New.
type Launcher struct{}

// New returns a ready Launcher.
func New() *Launcher { return &Launcher{} }

// Launch starts the application described by entry and returns the PID
// of the started process. The child is detached via Setsid, so it keeps
// running after the server exits. Stdin/stdout/stderr are nil and the
// caller does not wait on the child.
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
	return cmd.Process.Pid, nil
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
