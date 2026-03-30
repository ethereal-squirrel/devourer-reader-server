package metadata

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu                sync.Mutex
	queue             []chan struct{}
	processing        bool
	lastRequestTime   time.Time
	requestsInPeriod  int
	periodStart       time.Time
	requestsPerPeriod int
	periodDuration    time.Duration
	minInterval       time.Duration
}

func NewRateLimiter(requestsPerPeriod int, periodDuration, minInterval time.Duration) *RateLimiter {
	return &RateLimiter{
		requestsPerPeriod: requestsPerPeriod,
		periodDuration:    periodDuration,
		minInterval:       minInterval,
		periodStart:       time.Now(),
	}
}

func (r *RateLimiter) Wait() {
	ch := make(chan struct{})
	r.mu.Lock()
	r.queue = append(r.queue, ch)
	if !r.processing {
		r.processing = true
		go r.drain()
	}
	r.mu.Unlock()
	<-ch
}

func (r *RateLimiter) drain() {
	for {
		r.mu.Lock()
		if len(r.queue) == 0 {
			r.processing = false
			r.mu.Unlock()
			return
		}

		now := time.Now()

		if now.Sub(r.periodStart) >= r.periodDuration {
			r.requestsInPeriod = 0
			r.periodStart = now
		}

		if r.requestsInPeriod >= r.requestsPerPeriod {
			wait := r.periodStart.Add(r.periodDuration).Sub(now)
			r.mu.Unlock()
			time.Sleep(wait)
			continue
		}

		if !r.lastRequestTime.IsZero() {
			elapsed := now.Sub(r.lastRequestTime)
			if elapsed < r.minInterval {
				wait := r.minInterval - elapsed
				r.mu.Unlock()
				time.Sleep(wait)
				continue
			}
		}

		ch := r.queue[0]
		r.queue = r.queue[1:]
		r.lastRequestTime = time.Now()
		r.requestsInPeriod++
		r.mu.Unlock()

		close(ch)
	}
}

var (
	JikanLimiter       = NewRateLimiter(45, 60*time.Second, 400*time.Millisecond)
	GoogleBooksLimiter = NewRateLimiter(30, 60*time.Second, 400*time.Millisecond)
	OpenLibraryLimiter = NewRateLimiter(30, 60*time.Second, 400*time.Millisecond)
	ComicVineLimiter   = NewRateLimiter(200, 3600*time.Second, 400*time.Millisecond)
)
