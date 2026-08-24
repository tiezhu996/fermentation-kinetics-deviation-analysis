package middleware

import "time"

func (r *RateLimiter) allow(key string, now time.Time) bool {
	if window, exists := r.windows[key]; exists {
		return r.refresh(key, window, now)
	}
	r.windows[key] = rateWindow{start: now, count: 1}
	return true
}

func (r *RateLimiter) refresh(key string, window rateWindow, now time.Time) bool {
	if now.Sub(window.start) >= time.Minute {
		r.windows[key] = rateWindow{start: now, count: 1}
		return true
	}
	window.count++
	r.windows[key] = window
	if len(r.windows) > 10000 {
		r.sweep(now)
	}
	return window.count <= r.limit
}

func (r *RateLimiter) sweep(now time.Time) {
	for candidate, state := range r.windows {
		if now.Sub(state.start) > 2*time.Minute {
			delete(r.windows, candidate)
		}
	}
}
