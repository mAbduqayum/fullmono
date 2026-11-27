package auth

import (
	"math"
	"strings"
	"sync"
	"time"
)

const (
	// maxFailures failed logins from one address within failureWindow, and
	// it waits for the oldest to leave the window. bcrypt makes each guess
	// slow; this makes them few: 40 an hour, so a million-word list takes
	// almost 3 years.
	maxFailures   = 10
	failureWindow = 15 * time.Minute
	// pruneAbove addresses, and each new failure first drops those whose
	// failures have all left the window, so the map can't grow for ever.
	pruneAbove = 10_000
)

// failures counts each address's failed logins in this process's memory,
// which is enough for one api, as compose runs it. Replicas would each keep
// their own count, and need a shared one instead (Redis, or a table).
//
// Only failures count, so an admin who types the password right is never
// slowed; and they count by address, not by username, so someone guessing
// can't lock the real admin out.
type failures struct {
	mu     sync.Mutex
	byAddr map[string][]time.Time
	now    func() time.Time
}

func newFailures(now func() time.Time) *failures {
	return &failures{byAddr: map[string][]time.Time{}, now: now}
}

// wait says how long the address must wait before it tries again, 0 when it
// needn't.
func (f *failures) wait(addr string) time.Duration {
	f.mu.Lock()
	defer f.mu.Unlock()
	recent := f.recent(addr)
	if len(recent) < maxFailures {
		return 0
	}
	return recent[0].Add(failureWindow).Sub(f.now())
}

// add counts a failed login from the address.
func (f *failures) add(addr string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.byAddr) > pruneAbove {
		for other := range f.byAddr {
			f.recent(other)
		}
	}
	f.byAddr[addr] = append(f.recent(addr), f.now())
}

// forget drops the address's count: its login succeeded.
func (f *failures) forget(addr string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.byAddr, addr)
}

// recent is the address's failures inside the window, oldest first, and
// drops the older ones. The caller holds f.mu.
func (f *failures) recent(addr string) []time.Time {
	cutoff := f.now().Add(-failureWindow)
	times := f.byAddr[addr]
	for len(times) > 0 && !times[0].After(cutoff) {
		times = times[1:]
	}
	if len(times) == 0 {
		delete(f.byAddr, addr)
		return nil
	}
	f.byAddr[addr] = times
	return times
}

// visitor is the address a login comes from: the first in X-Forwarded-For,
// which Caddy sets to the visitor's own. Without the header (`just dev`'s
// Vite doesn't send one) every login shares one count.
func visitor(forwardedFor *string) string {
	if forwardedFor == nil {
		return ""
	}
	first, _, _ := strings.Cut(*forwardedFor, ",")
	return strings.TrimSpace(first)
}

// retryAfter is a wait in whole seconds, rounded up, for the Retry-After header.
func retryAfter(wait time.Duration) int {
	return int(math.Ceil(wait.Seconds()))
}
