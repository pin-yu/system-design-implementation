package rate_limiter

import (
	"context"
	"log"
	"sync"
	"time"
)

type RateLimiter interface {
	TryRequest() bool
	Reset()
	Close()
}

// token bucket algorithm
type TokenBucketRateLimiter struct {
	m1     sync.RWMutex
	cancel context.CancelFunc
	done   chan struct{}

	refillTokenPerSecond int32
	maxVolume            int32

	m2     sync.Mutex
	volume int32
}

func NewTokenBucketRateLimiter(refillTokenPerSecond, maxVolume int32) *TokenBucketRateLimiter {
	r := &TokenBucketRateLimiter{
		done:                 make(chan struct{}),
		refillTokenPerSecond: refillTokenPerSecond,
		maxVolume:            maxVolume,
	}
	r.Reset()
	return r
}

func (r *TokenBucketRateLimiter) TryRequest() bool {
	r.m2.Lock()
	defer r.m2.Unlock()
	if r.volume > 0 {
		r.volume--
		log.Printf("reduce one token from the bucket. token_num=%d", r.volume)
		return true
	}
	log.Println("the bucket is empty")
	return false
}

func (r *TokenBucketRateLimiter) Reset() {
	shouldWait := false
	r.m1.RLock()
	if r.cancel != nil {
		shouldWait = true
	}
	r.m1.RUnlock()

	r.Close()

	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		r.m1.Lock()
		r.cancel = cancel
		r.m1.Unlock()

		r.m2.Lock()
		r.volume = r.maxVolume
		r.m2.Unlock()

	loop:
		for {
			select {
			case <-time.After(1 * time.Second / time.Duration(r.refillTokenPerSecond)):
				r.m2.Lock()
				if r.volume < r.maxVolume {
					r.volume++
					log.Printf("add one token into the bucket, token_num=%d", r.volume)
				} else {
					log.Printf("the bucket is filled, token_num=%d", r.volume)
				}
				r.m2.Unlock()
			case <-ctx.Done():
				log.Print("stop ticker")
				break loop
			}
		}

		r.m1.Lock()
		r.cancel()
		r.cancel = nil
		r.m1.Unlock()

		r.done <- struct{}{}
	}()

	if shouldWait {
		<-r.done
	}

	log.Println("reset sucessfully")
}

func (r *TokenBucketRateLimiter) Close() {
	r.m1.Lock()
	if r.cancel != nil {
		r.cancel()
	}
	r.m1.Unlock()
}
