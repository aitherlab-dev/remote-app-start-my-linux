package terminal

import (
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

// Session manages a single interactive PTY session. It spawns the
// user's login shell (or /bin/bash as fallback) and exposes read /
// write / resize operations for use by a WebSocket relay.
type Session struct {
	cmd  *exec.Cmd
	ptmx *os.File
	once sync.Once
}

// NewSession starts a new PTY session running the user's shell.
func NewSession() (*Session, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	cmd := exec.Command(shell, "-l")
	cmd.Env = os.Environ()
	// Start in a new process group so we can signal the whole tree.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	// Set a reasonable default size.
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

	return &Session{cmd: cmd, ptmx: ptmx}, nil
}

// Read reads from the PTY master (i.e. the shell's output).
func (s *Session) Read(buf []byte) (int, error) {
	return s.ptmx.Read(buf)
}

// Write writes to the PTY master (i.e. the shell's input).
func (s *Session) Write(data []byte) (int, error) {
	return s.ptmx.Write(data)
}

// Resize changes the PTY window size.
func (s *Session) Resize(cols, rows uint16) error {
	return pty.Setsize(s.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
}

// Close terminates the PTY session: sends SIGHUP to the process
// group and closes the master fd. Safe to call multiple times.
func (s *Session) Close() {
	s.once.Do(func() {
		// Signal the whole process group (negative PID).
		if s.cmd.Process != nil {
			_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGHUP)
		}
		_ = s.ptmx.Close()
		// Reap the child — ignore errors (may already have exited).
		_ = s.cmd.Wait()
	})
}
