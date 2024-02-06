package rate_limiter

import "sync"

type RateLimiterMap struct {
	m            sync.RWMutex
	rateLimiters map[string]RateLimiter // key -> rate limiter
}

func NewRateLimiterMap() *RateLimiterMap {
	return &RateLimiterMap{
		rateLimiters: map[string]RateLimiter{},
	}
}

func (r *RateLimiterMap) Add(k string, v RateLimiter) {
	r.m.Lock()
	defer r.m.Unlock()
	r.rateLimiters[k] = v
}

func (r *RateLimiterMap) Get(k string) RateLimiter {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.rateLimiters[k]
}
