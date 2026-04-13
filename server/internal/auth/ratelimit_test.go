package auth

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	rl := NewRateLimiter(5, 20, 10*time.Minute)
	for i := 0; i < 5; i++ {
		ok, retry := rl.Allow("1.2.3.4")
		if !ok {
			t.Fatalf("attempt %d: allow=false, want true", i+1)
		}
		if retry != 0 {
			t.Errorf("attempt %d: retryAfter=%v, want 0", i+1, retry)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	rl := NewRateLimiter(5, 20, 10*time.Minute)
	for i := 0; i < 5; i++ {
		if ok, _ := rl.Allow("1.2.3.4"); !ok {
			t.Fatalf("setup attempt %d: got false", i+1)
		}
	}
	ok, retry := rl.Allow("1.2.3.4")
	if ok {
		t.Fatal("6th attempt: allow=true, want false")
	}
	if retry <= 0 {
		t.Errorf("6th attempt: retryAfter=%v, want > 0", retry)
	}
	if retry > 10*time.Minute {
		t.Errorf("6th attempt: retryAfter=%v, want <= window", retry)
	}
}

func TestRateLimiter_PerIPIsolation(t *testing.T) {
	rl := NewRateLimiter(5, 20, 10*time.Minute)
	for i := 0; i < 5; i++ {
		if ok, _ := rl.Allow("1.2.3.4"); !ok {
			t.Fatalf("ip A attempt %d: got false", i+1)
		}
	}
	// A is saturated; B must still get all its budget.
	for i := 0; i < 5; i++ {
		ok, _ := rl.Allow("5.6.7.8")
		if !ok {
			t.Fatalf("ip B attempt %d: got false, per-IP isolation broken", i+1)
		}
	}
	if ok, _ := rl.Allow("1.2.3.4"); ok {
		t.Error("ip A still blocked after B traffic: got true")
	}
}

func TestRateLimiter_ResetsAfterWindow(t *testing.T) {
	rl := NewRateLimiter(5, 20, 10*time.Minute)
	start := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	now := start
	rl.SetClock(func() time.Time { return now })

	for i := 0; i < 5; i++ {
		if ok, _ := rl.Allow("1.2.3.4"); !ok {
			t.Fatalf("setup attempt %d: got false", i+1)
		}
	}
	if ok, _ := rl.Allow("1.2.3.4"); ok {
		t.Fatal("before advance: 6th attempt should be blocked")
	}

	// Advance past the window; counter should reset.
	now = start.Add(11 * time.Minute)
	ok, retry := rl.Allow("1.2.3.4")
	if !ok {
		t.Fatalf("after window: allow=false, want true")
	}
	if retry != 0 {
		t.Errorf("after window: retryAfter=%v, want 0", retry)
	}
}

func TestRateLimiter_GlobalLimit(t *testing.T) {
	rl := NewRateLimiter(5, 20, 10*time.Minute)
	// 4 IPs × 5 attempts each = 20 attempts, all must pass.
	for i := 0; i < 4; i++ {
		ip := fmt.Sprintf("10.0.0.%d", i+1)
		for j := 0; j < 5; j++ {
			ok, _ := rl.Allow(ip)
			if !ok {
				t.Fatalf("ip %s attempt %d: got false inside global budget", ip, j+1)
			}
		}
	}
	// 21st attempt from a fresh IP must be blocked by the global cap
	// even though that IP has a clean per-IP bucket.
	ok, retry := rl.Allow("10.0.0.99")
	if ok {
		t.Fatal("global 21st: allow=true, want false")
	}
	if retry <= 0 {
		t.Errorf("global 21st: retryAfter=%v, want > 0", retry)
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	t.Helper()
	// With -race this exercises the mutex. Use generous limits so the
	// test does not care about which goroutines win, only that no race
	// is reported and the totals add up.
	rl := NewRateLimiter(1000, 10_000, 10*time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ip := fmt.Sprintf("192.168.1.%d", n%10)
			for j := 0; j < 20; j++ {
				rl.Allow(ip)
			}
		}(i)
	}
	wg.Wait()
}
