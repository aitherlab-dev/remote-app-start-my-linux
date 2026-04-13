package launcher

import (
	"context"
	"errors"
	"sync"
	"syscall"
	"time"
)

// Tracker keeps track of PIDs started per application id so callers
// can ask whether an app is currently running. Liveness is probed via
// syscall.Kill(pid, 0).
//
// Note: PID reuse is a known limitation — after a process exits, its
// PID may be recycled by an unrelated process. For MVP we accept that
// Alive may briefly report true for a recycled PID. A more robust
// implementation would track start times via /proc.
type Tracker struct {
	mu   sync.Mutex
	pids map[string][]int
}

// NewTracker returns a Tracker with an empty state.
func NewTracker() *Tracker {
	return &Tracker{pids: make(map[string][]int)}
}

// Register associates pid with id. Duplicate PIDs are allowed and will
// be collapsed naturally during Cleanup (they share the same liveness).
func (t *Tracker) Register(id string, pid int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pids[id] = append(t.pids[id], pid)
}

// Forget removes a single pid from id's list. If the list becomes
// empty the key is deleted from the map. Calling Forget with an
// unknown id or a pid that is not tracked is a no-op.
func (t *Tracker) Forget(id string, pid int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	src, ok := t.pids[id]
	if !ok {
		return
	}
	for i, p := range src {
		if p == pid {
			src = append(src[:i], src[i+1:]...)
			break
		}
	}
	if len(src) == 0 {
		delete(t.pids, id)
		return
	}
	t.pids[id] = src
}

// Alive reports whether at least one registered PID for id is alive.
// It prunes dead PIDs for that id before answering; if none remain,
// the key is removed from the map.
func (t *Tracker) Alive(id string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cleanupLocked(id)
}

// Pids returns a copy of the live PIDs for id (a fresh slice that the
// caller may mutate without affecting internal state).
func (t *Tracker) Pids(id string) []int {
	t.mu.Lock()
	defer t.mu.Unlock()
	src := t.pids[id]
	if len(src) == 0 {
		return nil
	}
	out := make([]int, 0, len(src))
	for _, pid := range src {
		if isAlive(pid) {
			out = append(out, pid)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// Cleanup walks every tracked id, drops dead PIDs, and removes ids
// that no longer have any live PIDs. Safe to call repeatedly.
func (t *Tracker) Cleanup() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for id := range t.pids {
		t.cleanupLocked(id)
	}
}

// Snapshot returns a deep copy of the current id → live PIDs mapping.
// Mutating the returned map or its slices does not affect the Tracker.
func (t *Tracker) Snapshot() map[string][]int {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make(map[string][]int, len(t.pids))
	for id, src := range t.pids {
		live := make([]int, 0, len(src))
		for _, pid := range src {
			if isAlive(pid) {
				live = append(live, pid)
			}
		}
		if len(live) > 0 {
			out[id] = live
		}
	}
	return out
}

// CleanupLoop runs Cleanup on interval until ctx is Done. It is meant
// to be launched in its own goroutine.
func (t *Tracker) CleanupLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.Cleanup()
		}
	}
}

// cleanupLocked drops dead PIDs for id and deletes the key if the
// slice becomes empty. Returns whether any live PID remained. Caller
// must hold t.mu.
func (t *Tracker) cleanupLocked(id string) bool {
	src, ok := t.pids[id]
	if !ok {
		return false
	}
	live := src[:0]
	for _, pid := range src {
		if isAlive(pid) {
			live = append(live, pid)
		}
	}
	if len(live) == 0 {
		delete(t.pids, id)
		return false
	}
	t.pids[id] = live
	return true
}

// isAlive probes process existence via kill(pid, 0). A nil error means
// the signal could be delivered (process alive); EPERM means the
// process exists but belongs to another user (still alive); ESRCH or
// any other error is treated as dead.
func isAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	if errors.Is(err, syscall.EPERM) {
		return true
	}
	return false
}
