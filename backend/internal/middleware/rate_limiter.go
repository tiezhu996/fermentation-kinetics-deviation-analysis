package middleware

import "time"

// allow reports whether a request under key is admitted at now. When the
// request is denied, resetIn is the time until the current window resets and
// the caller may start a fresh count; it is only meaningful when allowed is
// false. All access to r.windows is serialized under r.mu, which is what makes
// the per-key counter safe under concurrent logins.
func (r *RateLimiter) allow(key string, now time.Time) (allowed bool, resetIn time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	window, exists := r.windows[key]
	if !exists {
		r.windows[key] = rateWindow{start: now, count: 1}
		return true, 0
	}

	if now.Sub(window.start) >= time.Minute {
		r.windows[key] = rateWindow{start: now, count: 1}
		return true, 0
	}

	window.count++
	r.windows[key] = window
	if len(r.windows) > 10000 {
		r.sweepLocked(now)
	}

	if window.count <= r.limit {
		return true, 0
	}
	return false, time.Minute - now.Sub(window.start)
}

// sweepLocked evicts stale windows. Callers must already hold r.mu; it only
// touches r.windows, so it needs no locking of its own.
func (r *RateLimiter) sweepLocked(now time.Time) {
	for candidate, state := range r.windows {
		if now.Sub(state.start) > 2*time.Minute {
			delete(r.windows, candidate)
		}
	}
}
