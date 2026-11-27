package ratelimiter

import (
	"sync"
	"time"
)

// Limiter — fixed-window лимитер. Окно сбрасывается само по времени,
// отдельные тикеры сброса не нужны.
type Limiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	start  time.Time
	count  int
}

func New(limit int, window time.Duration) *Limiter {
	return &Limiter{limit: limit, window: window, start: time.Now()}
}

func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if now.Sub(l.start) >= l.window {
		l.start = now
		l.count = 0
	}
	if l.count >= l.limit {
		return false
	}
	l.count++
	return true
}
