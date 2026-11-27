package auth

import (
	"testing"
	"time"
)

// clock is a time the test moves by hand.
type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func TestFailuresWaitOnlyOnceThereAreEnough(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)}
	f := newFailures(c.now)
	for range maxFailures - 1 {
		f.add("203.0.113.7")
		c.t = c.t.Add(time.Minute)
	}
	if wait := f.wait("203.0.113.7"); wait != 0 {
		t.Fatalf("after %d failures: wait %s", maxFailures-1, wait)
	}
	f.add("203.0.113.7")
	// The first failure was 9 minutes ago: it leaves the window in 6.
	if wait := f.wait("203.0.113.7"); wait != 6*time.Minute {
		t.Errorf("after %d: wait %s, want 6m", maxFailures, wait)
	}
	if wait := f.wait("198.51.100.1"); wait != 0 {
		t.Errorf("another address: wait %s", wait)
	}
	c.t = c.t.Add(6 * time.Minute)
	if wait := f.wait("203.0.113.7"); wait != 0 {
		t.Errorf("once the first left the window: wait %s", wait)
	}
}

func TestFailuresForgetASuccess(t *testing.T) {
	c := &clock{t: time.Now()}
	f := newFailures(c.now)
	for range maxFailures {
		f.add("203.0.113.7")
	}
	f.forget("203.0.113.7")
	if wait := f.wait("203.0.113.7"); wait != 0 {
		t.Errorf("wait %s", wait)
	}
}

func TestFailuresDropOldAddresses(t *testing.T) {
	c := &clock{t: time.Now()}
	f := newFailures(c.now)
	f.add("203.0.113.7")
	c.t = c.t.Add(failureWindow)
	f.wait("203.0.113.7")
	if len(f.byAddr) != 0 {
		t.Errorf("kept %v", f.byAddr)
	}
}

func TestVisitorIsTheFirstForwardedAddress(t *testing.T) {
	for header, want := range map[string]string{
		"203.0.113.7":                 "203.0.113.7",
		"203.0.113.7, 198.51.100.1":   "203.0.113.7",
		" 2001:db8::1 , 198.51.100.1": "2001:db8::1",
	} {
		if got := visitor(&header); got != want {
			t.Errorf("visitor(%q) = %q, want %q", header, got, want)
		}
	}
	if got := visitor(nil); got != "" {
		t.Errorf("no header: %q", got)
	}
}

func TestRetryAfterRoundsUp(t *testing.T) {
	if got := retryAfter(1500 * time.Millisecond); got != 2 {
		t.Errorf("1.5s: %d", got)
	}
}
