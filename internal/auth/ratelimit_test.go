package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newTestLimiter() *ipLimiter {
	return &ipLimiter{attempts: make(map[string][]time.Time)}
}

func TestIPLimiter_AllowsUnderLimit(t *testing.T) {
	l := newTestLimiter()
	for i := 0; i < rateLimitMaxReqs-1; i++ {
		assert.True(t, l.allow("1.2.3.4"), "call %d should be allowed", i+1)
	}
}

func TestIPLimiter_BlocksAtLimit(t *testing.T) {
	l := newTestLimiter()
	for i := 0; i < rateLimitMaxReqs; i++ {
		l.allow("1.2.3.4")
	}
	assert.False(t, l.allow("1.2.3.4"), "request beyond limit should be blocked")
}

func TestIPLimiter_DifferentIPsAreIndependent(t *testing.T) {
	l := newTestLimiter()

	for i := 0; i < rateLimitMaxReqs; i++ {
		l.allow("1.1.1.1")
	}

	assert.True(t, l.allow("2.2.2.2"))
}

func TestIPLimiter_ExpiresOldAttempts(t *testing.T) {
	l := newTestLimiter()

	old := time.Now().Add(-(rateLimitWindow + time.Second))
	stale := make([]time.Time, rateLimitMaxReqs)
	for i := range stale {
		stale[i] = old
	}
	l.attempts["1.2.3.4"] = stale

	assert.True(t, l.allow("1.2.3.4"), "old attempts should be pruned and request allowed")
}

func TestIPLimiter_MixedFreshAndStale(t *testing.T) {
	l := newTestLimiter()
	old := time.Now().Add(-(rateLimitWindow + time.Second))

	for i := 0; i < rateLimitMaxReqs-1; i++ {
		l.attempts["1.2.3.4"] = append(l.attempts["1.2.3.4"], old)
	}
	for i := 0; i < rateLimitMaxReqs-1; i++ {
		l.allow("1.2.3.4")
	}
	assert.True(t, l.allow("1.2.3.4"))
}
