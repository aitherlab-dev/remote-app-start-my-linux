package launcher

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// ErrUnknownTerminal is returned by LaunchCommand when the requested
// terminal emulator is not one of the supported profiles. The error
// message names the failed terminal so the web UI can surface a
// readable toast.
var ErrUnknownTerminal = errors.New("unknown terminal emulator")

// termProfile describes how to build an argv for a given terminal
// emulator so that it opens a new window and runs a shell payload.
// execFlag is the CLI flag each terminal uses to introduce the
// command that follows (most want -e, gnome-terminal wants --, kitty
// and foot just take the command as trailing positional args).
type termProfile struct {
	execFlag string
}

// supportedTerminals is the whitelist of terminal emulators the
// launcher knows how to drive. Keeping this as a finite map keeps
// the attack surface small — a user-supplied terminal name goes
// through this lookup before ever being handed to exec.
var supportedTerminals = map[string]termProfile{
	"kitty":          {execFlag: ""},
	"ghostty":        {execFlag: "-e"},
	"alacritty":      {execFlag: "-e"},
	"foot":           {execFlag: ""},
	"konsole":        {execFlag: "-e"},
	"gnome-terminal": {execFlag: "--"},
	"xfce4-terminal": {execFlag: "-e"},
	"xterm":          {execFlag: "-e"},
}

// SupportedTerminals returns the sorted list of terminal emulator
// names LaunchCommand understands. The web UI uses this to populate
// the dropdown on the shortcut-edit form.
func SupportedTerminals() []string {
	out := make([]string, 0, len(supportedTerminals))
	for name := range supportedTerminals {
		out = append(out, name)
	}
	// Stable ordering for a deterministic UI.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// LaunchCommand starts a user-defined shortcut command inside a
// terminal emulator, detached from the server, and registers the
// resulting pid with the Tracker under id so /api/apps can report
// Running=true while the window is open.
//
// terminal is the emulator to use. A blank value falls back to
// defaultTerminal; a blank defaultTerminal after that triggers the
// first-available scan via findTerminal() (same heuristic Terminal=
// entries use). An unknown terminal name returns ErrUnknownTerminal.
//
// cwd and command are joined into a single shell payload of the
// shape `cd '<cwd>' && exec <command>` — using shell lets the user
// write pipes, env vars and glob patterns in the shortcut command
// without the server having to parse them.
func (l *Launcher) LaunchCommand(id, terminal, defaultTerminal, cwd, command string) (int, error) {
	if strings.TrimSpace(command) == "" {
		return 0, fmt.Errorf("%w: command is empty", ErrEmptyExec)
	}

	term := strings.TrimSpace(terminal)
	if term == "" {
		term = strings.TrimSpace(defaultTerminal)
	}
	var argv []string
	switch term {
	case "":
		// No preference anywhere — pick the first one on PATH from the
		// legacy terminalCandidates list. Wrap the payload with sh -c
		// using a generic -e flag (xterm-style). Every terminal in
		// terminalCandidates speaks -e.
		path, err := findTerminal()
		if err != nil {
			return 0, err
		}
		argv = []string{path, "-e", "sh", "-c", buildPayload(cwd, command)}
	default:
		prof, ok := supportedTerminals[term]
		if !ok {
			return 0, fmt.Errorf("%w: %q", ErrUnknownTerminal, term)
		}
		path, err := exec.LookPath(term)
		if err != nil {
			return 0, fmt.Errorf("terminal %q not found on PATH: %w", term, err)
		}
		argv = []string{path}
		if prof.execFlag != "" {
			argv = append(argv, prof.execFlag)
		}
		argv = append(argv, "sh", "-c", buildPayload(cwd, command))
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start shortcut %q: %w", id, err)
	}
	pid := cmd.Process.Pid
	l.tracker.Register(id, pid)

	// Reap so the terminal emulator window isn't left as a zombie
	// after it closes, and so Alive flips back to false promptly.
	go func(cmd *exec.Cmd, id string, pid int) {
		_ = cmd.Wait()
		l.tracker.Forget(id, pid)
	}(cmd, id, pid)

	return pid, nil
}

// buildPayload produces the single string passed to `sh -c` that
// first changes into cwd (if set) and then exec's the user command.
// exec replaces the shell process so cmd.Wait correctly tracks the
// lifetime of the user's program rather than of the outer sh.
func buildPayload(cwd, command string) string {
	command = strings.TrimSpace(command)
	if cwd = strings.TrimSpace(cwd); cwd == "" {
		return "exec " + command
	}
	return "cd " + shellSingleQuote(cwd) + " && exec " + command
}

// shellSingleQuote wraps s in POSIX single quotes, escaping any
// embedded single quotes by closing the literal, emitting \', and
// reopening. This is the canonical way to safely embed arbitrary
// text inside a shell command line.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, `'`, `'\''`) + "'"
}
