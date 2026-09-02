package hsm

import (
	"context"
	"crypto"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// CircuitBreaker protects HSM from cascading failures (single USB device).
// States: closed (normal), open (failing), half-open (probe).
type CircuitBreaker struct {
	mu               sync.Mutex
	failures         int
	threshold        int
	resetTimeout     time.Duration
	lastFailure      time.Time
	state            string // closed, open, half-open
}

func NewCircuitBreaker(threshold int, resetTimeout time.Duration) *CircuitBreaker {
	if threshold == 0 {
		threshold = 5
	}
	if resetTimeout == 0 {
		resetTimeout = 30 * time.Second
	}
	return &CircuitBreaker{threshold: threshold, resetTimeout: resetTimeout, state: "closed"}
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.state {
	case "open":
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = "half-open"
			return true
		}
		return false
	case "half-open":
		return true
	default: // closed
		return true
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = "closed"
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= cb.threshold {
		cb.state = "open"
	}
}

// RateLimiter protects HSM from rate_limited errors (429). Use per-backend.
type RateLimiter struct {
	limiter *rate.Limiter
}

func NewRateLimiter(rps int, burst int) *RateLimiter {
	if rps <= 0 {
		rps = 10
	}
	if burst <= 0 {
		burst = rps
	}
	return &RateLimiter{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
}

func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.limiter.Wait(ctx)
}

// ResilientDriver wraps Driver with circuit breaker + rate limiting + timeout.
type resilientDriver struct {
	next    Driver
	cb      *CircuitBreaker
	limiter *RateLimiter
	timeout time.Duration
}

func NewResilientDriver(next Driver, cb *CircuitBreaker, limiter *RateLimiter, timeout time.Duration) Driver {
	return &resilientDriver{next: next, cb: cb, limiter: limiter, timeout: timeout}
}

func (r *resilientDriver) withResilience(ctx context.Context, op func(context.Context) error) error {
	if r.limiter != nil {
		if err := r.limiter.Wait(ctx); err != nil {
			return err
		}
	}
	if r.cb != nil && !r.cb.Allow() {
		return ErrDeviceNotReady
	}
	// Timeout per op
	if r.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.timeout)
		defer cancel()
	}
	err := op(ctx)
	if r.cb != nil {
		if err != nil {
			r.cb.RecordFailure()
		} else {
			r.cb.RecordSuccess()
		}
	}
	return err
}

func (r *resilientDriver) GenerateKey(ctx context.Context, spec KeySpec) (*KeyInfo, error) {
	var ki *KeyInfo
	err := r.withResilience(ctx, func(c context.Context) error {
		var e error
		ki, e = r.next.GenerateKey(c, spec)
		return e
	})
	return ki, err
}

func (r *resilientDriver) GetPublicKey(ctx context.Context, id KeyID) (crypto.PublicKey, string, error) {
	return r.next.GetPublicKey(ctx, id)
}

func (r *resilientDriver) Sign(ctx context.Context, id KeyID, digest []byte, mech Mechanism) ([]byte, error) {
	var sig []byte
	err := r.withResilience(ctx, func(c context.Context) error {
		var e error
		sig, e = r.next.Sign(c, id, digest, mech)
		return e
	})
	return sig, err
}

func (r *resilientDriver) Signer(ctx context.Context, id KeyID, mech Mechanism) (Signer, error) {
	return r.next.Signer(ctx, id, mech)
}

func (r *resilientDriver) DeleteKey(ctx context.Context, id KeyID) error {
	return r.withResilience(ctx, func(c context.Context) error { return r.next.DeleteKey(c, id) })
}

func (r *resilientDriver) ListKeys(ctx context.Context) ([]KeyInfo, error) {
	var keys []KeyInfo
	err := r.withResilience(ctx, func(c context.Context) error {
		var e error
		keys, e = r.next.ListKeys(c)
		return e
	})
	return keys, err
}

func (r *resilientDriver) Info(ctx context.Context) (*SlotInfo, error) {
	return r.next.Info(ctx)
}

func (r *resilientDriver) Close() error { return r.next.Close() }
