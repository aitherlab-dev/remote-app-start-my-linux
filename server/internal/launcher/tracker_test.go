package launcher

import (
	"context"
	"os/exec"
	"sort"
	"sync"
	"syscall"
	"testing"
	"time"
)

// startSleep starts /bin/sh -c "sleep N" without Setsid (plain test
// child) and registers cleanup: SIGKILL if still alive, then Wait so
// no zombie is left behind.
func startSleep(t *testing.T, seconds string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("/bin/sh", "-c", "sleep "+seconds)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	return cmd
}

// killAndReap sends SIGKILL and Waits so the child is fully reaped
// before the test proceeds. Safe to call multiple times.
func killAndReap(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
}

func TestTracker_RegisterAndAliveOnLiveProcess(t *testing.T) {
	cmd := startSleep(t, "1")
	tr := NewTracker()
	tr.Register("test", cmd.Process.Pid)

	if !tr.Alive("test") {
		t.Fatalf("Alive(test) = false just after Register, want true")
	}

	// Wait until the sleep dies, then reap to flip kill(pid,0) to ESRCH.
	if err := cmd.Wait(); err != nil {
		// sleep exits 0; Wait err is surfaced for visibility but not fatal.
		t.Logf("wait: %v", err)
	}

	if tr.Alive("test") {
		t.Errorf("Alive(test) = true after process exit, want false")
	}
	// Key must be pruned.
	if _, ok := tr.Snapshot()["test"]; ok {
		t.Errorf("Snapshot still contains dead id %q", "test")
	}
}

func TestTracker_AliveReturnsFalseForUnknown(t *testing.T) {
	tr := NewTracker()
	if tr.Alive("nonexistent") {
		t.Errorf("Alive(nonexistent) = true, want false")
	}
}

func TestTracker_AliveReturnsFalseForDeadPID(t *testing.T) {
	tr := NewTracker()
	tr.Register("test", 999999)
	if tr.Alive("test") {
		t.Errorf("Alive(test) = true for bogus pid, want false")
	}
}

func TestTracker_MultipleInstancesOneAliveOneDead(t *testing.T) {
	first := startSleep(t, "10")
	second := startSleep(t, "10")
	tr := NewTracker()
	tr.Register("app", first.Process.Pid)
	tr.Register("app", second.Process.Pid)

	// Kill the first and reap it so kill(pid,0) returns ESRCH.
	killAndReap(t, first)

	if !tr.Alive("app") {
		t.Errorf("Alive(app) = false with one live pid remaining, want true")
	}
	live := tr.Pids("app")
	if len(live) != 1 || live[0] != second.Process.Pid {
		t.Errorf("Pids(app) = %v, want [%d]", live, second.Process.Pid)
	}

	killAndReap(t, second)
	if tr.Alive("app") {
		t.Errorf("Alive(app) = true after both pids dead, want false")
	}
}

func TestTracker_Cleanup(t *testing.T) {
	live := startSleep(t, "10")
	tr := NewTracker()
	tr.Register("mixed", live.Process.Pid)
	tr.Register("mixed", 999999)

	tr.Cleanup()

	got := tr.Pids("mixed")
	if len(got) != 1 || got[0] != live.Process.Pid {
		t.Errorf("after Cleanup Pids(mixed) = %v, want [%d]", got, live.Process.Pid)
	}

	tr2 := NewTracker()
	tr2.Register("dead-only", 999998)
	tr2.Cleanup()
	if pids := tr2.Pids("dead-only"); pids != nil {
		t.Errorf("Pids(dead-only) = %v, want nil", pids)
	}
	if _, ok := tr2.Snapshot()["dead-only"]; ok {
		t.Errorf("Snapshot still contains dead id %q", "dead-only")
	}
}

func TestTracker_Snapshot(t *testing.T) {
	a1 := startSleep(t, "10")
	a2 := startSleep(t, "10")
	b1 := startSleep(t, "10")

	tr := NewTracker()
	tr.Register("a", a1.Process.Pid)
	tr.Register("a", a2.Process.Pid)
	tr.Register("b", b1.Process.Pid)

	snap := tr.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("Snapshot keys = %d, want 2", len(snap))
	}
	aPids := append([]int(nil), snap["a"]...)
	sort.Ints(aPids)
	wantA := []int{a1.Process.Pid, a2.Process.Pid}
	sort.Ints(wantA)
	if len(aPids) != 2 || aPids[0] != wantA[0] || aPids[1] != wantA[1] {
		t.Errorf("snap[a] = %v, want %v", aPids, wantA)
	}
	if len(snap["b"]) != 1 || snap["b"][0] != b1.Process.Pid {
		t.Errorf("snap[b] = %v, want [%d]", snap["b"], b1.Process.Pid)
	}

	// Mutating the returned snapshot must not leak into the tracker.
	snap["a"] = append(snap["a"], 12345)
	snap["ghost"] = []int{7}

	again := tr.Snapshot()
	if len(again) != 2 {
		t.Errorf("second Snapshot has %d keys, want 2", len(again))
	}
	if len(again["a"]) != 2 {
		t.Errorf("second snap[a] len = %d, want 2", len(again["a"]))
	}
	if _, ok := again["ghost"]; ok {
		t.Errorf("second Snapshot contains ghost key, mutation leaked")
	}
}

func TestTracker_PidsReturnsCopy(t *testing.T) {
	a := startSleep(t, "10")
	b := startSleep(t, "10")
	tr := NewTracker()
	tr.Register("a", a.Process.Pid)
	tr.Register("a", b.Process.Pid)

	pids := tr.Pids("a")
	if len(pids) != 2 {
		t.Fatalf("Pids(a) len = %d, want 2", len(pids))
	}
	pids[0] = 999

	again := tr.Pids("a")
	for _, pid := range again {
		if pid == 999 {
			t.Errorf("internal state was mutated via returned slice: %v", again)
		}
	}
}

func TestTracker_ConcurrentAccess(t *testing.T) {
	tr := NewTracker()

	const n = 5
	cmds := make([]*exec.Cmd, n)
	for i := 0; i < n; i++ {
		cmds[i] = startSleep(t, "10")
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// Writer goroutines: register pids under different ids.
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := "app-" + string(rune('A'+i))
			for j := 0; j < 50; j++ {
				if ctx.Err() != nil {
					return
				}
				tr.Register(id, cmds[i].Process.Pid)
			}
		}(i)
	}

	// Reader goroutines: Alive / Pids / Snapshot.
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := "app-" + string(rune('A'+i))
			for j := 0; j < 50; j++ {
				if ctx.Err() != nil {
					return
				}
				_ = tr.Alive(id)
				_ = tr.Pids(id)
				_ = tr.Snapshot()
			}
		}(i)
	}

	// Cleanup goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				tr.Cleanup()
				time.Sleep(time.Millisecond)
			}
		}
	}()

	// Let the storm run briefly, then stop.
	time.Sleep(100 * time.Millisecond)
	cancel()
	wg.Wait()
}

func TestTracker_CleanupLoop(t *testing.T) {
	live := startSleep(t, "10")
	tr := NewTracker()
	tr.Register("app", live.Process.Pid)
	tr.Register("app", 999997)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		tr.CleanupLoop(ctx, 20*time.Millisecond)
		close(done)
	}()

	time.Sleep(150 * time.Millisecond)

	got := tr.Pids("app")
	if len(got) != 1 || got[0] != live.Process.Pid {
		t.Errorf("after CleanupLoop Pids(app) = %v, want [%d]", got, live.Process.Pid)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("CleanupLoop did not stop after ctx cancel")
	}
}

func TestTracker_IsAliveGuards(t *testing.T) {
	if isAlive(0) {
		t.Errorf("isAlive(0) = true, want false")
	}
	if isAlive(-1) {
		t.Errorf("isAlive(-1) = true, want false")
	}
	// Signal 0 to our own PID must succeed — we are alive.
	if !isAlive(syscall.Getpid()) {
		t.Errorf("isAlive(self) = false, want true")
	}
}
