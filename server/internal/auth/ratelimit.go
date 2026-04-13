package auth

import (
	"sync"
	"time"
)

// RateLimiter implements a simple fixed-window rate limiter that tracks
// attempts per client IP alongside a global counter. It is designed for
// the /api/pair endpoint: both a single attacker and a distributed
// spray attack must be slowed down, so Allow only returns true when
// both the per-IP and the global budget still have room.
//
// The limiter is safe for concurrent use. Time is read through the
// injectable `now` function so tests can advance a virtual clock via
// SetClock without sleeping.
type RateLimiter struct {
	mu        sync.Mutex
	perIP     map[string]*bucket
	global    *bucket
	perIPMax  int
	globalMax int
	window    time.Duration
	now       func() time.Time
}

// bucket is a fixed-window counter: `count` attempts seen since a
// window opened at `windowEnd - window`, where `windowEnd` marks the
// moment the current window expires. When the clock passes windowEnd
// the bucket resets on the next access.
type bucket struct {
	count     int
	windowEnd time.Time
}

// NewRateLimiter returns a limiter that allows up to perIPMax attempts
// per IP and globalMax attempts in total per rolling window of the
// given duration. The default clock is time.Now; SetClock can replace
// it for deterministic tests.
func NewRateLimiter(perIPMax, globalMax int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		perIP:     make(map[string]*bucket),
		global:    &bucket{},
		perIPMax:  perIPMax,
		globalMax: globalMax,
		window:    window,
		now:       time.Now,
	}
}

// SetClock installs a custom clock function. Tests use this to freeze
// or advance time without waiting. Production code should leave the
// default time.Now in place.
func (r *RateLimiter) SetClock(fn func() time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.now = fn
}

// Allow reports whether a request from ip is within the configured
// per-IP and global budgets. On success the per-IP and global counters
// are both incremented and (true, 0) is returned. On failure the
// second value is the duration until the earliest counter rolls over,
// so the caller can surface it as a Retry-After header.
func (r *RateLimiter) Allow(ip string) (bool, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()

	b, ok := r.perIP[ip]
	if !ok {
		b = &bucket{}
		r.perIP[ip] = b
	}
	if b.windowEnd.IsZero() || !b.windowEnd.After(now) {
		b.count = 0
		b.windowEnd = now.Add(r.window)
	}
	if r.global.windowEnd.IsZero() || !r.global.windowEnd.After(now) {
		r.global.count = 0
		r.global.windowEnd = now.Add(r.window)
	}

	if b.count >= r.perIPMax {
		return false, b.windowEnd.Sub(now)
	}
	if r.global.count >= r.globalMax {
		return false, r.global.windowEnd.Sub(now)
	}

	b.count++
	r.global.count++
	return true, 0
}
