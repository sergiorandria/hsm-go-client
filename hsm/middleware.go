package hsm

import (
	"context"
	"crypto"
	"errors"
	"log/slog"
	"time"
)

// middlewareDriver decorates a Driver with logging, metrics, retry (Decorator pattern).
// Keeps backends pure; cross-cutting concerns here for production.
type middlewareDriver struct {
	next Driver
	opts *options
}

func wrapWithMiddleware(d Driver, opts *options) Driver {
	if opts == nil {
		return d
	}
	// Only wrap if any middleware configured
	if opts.logger == nil && opts.metricsHook == nil && opts.maxRetries == 0 {
		return d
	}
	return &middlewareDriver{next: d, opts: opts}
}

func (m *middlewareDriver) log(op string, keyID KeyID, err error, dur time.Duration) {
	if m.opts.logger != nil {
		level := slog.LevelInfo
		if err != nil {
			level = slog.LevelError
		}
		m.opts.logger.Log(context.Background(), level, "hsm."+op,
			"key", keyID.Label,
			"duration_ms", dur.Milliseconds(),
			"error", err,
		)
	}
	if m.opts.metricsHook != nil {
		m.opts.metricsHook(op, dur, err)
	}
}

func (m *middlewareDriver) withRetry(ctx context.Context, op string, fn func() error) error {
	var lastErr error
	attempts := m.opts.maxRetries + 1
	for i := 0; i < attempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			// Retry only on transient
			if !isTransient(err) || i == attempts-1 {
				break
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(m.opts.retryDelay):
			}
		}
	}
	return lastErr
}

func isTransient(err error) bool {
	return err != nil && (errors.Is(err, ErrDeviceNotReady) || errors.Is(err, ErrTimeout))
}

func (m *middlewareDriver) GenerateKey(ctx context.Context, spec KeySpec) (*KeyInfo, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	start := time.Now()
	var ki *KeyInfo
	var err error
	retryErr := m.withRetry(ctx, "GenerateKey", func() error {
		ki, err = m.next.GenerateKey(ctx, spec)
		return err
	})
	dur := time.Since(start)
	m.log("GenerateKey", KeyID{Label: spec.Label}, err, dur)
	if retryErr != nil {
		return nil, err
	}
	return ki, err
}

func (m *middlewareDriver) GetPublicKey(ctx context.Context, id KeyID) (crypto.PublicKey, string, error) {
	if err := id.Validate(); err != nil {
		return nil, "", err
	}
	start := time.Now()
	pub, pem, err := m.next.GetPublicKey(ctx, id)
	m.log("GetPublicKey", id, err, time.Since(start))
	return pub, pem, err
}

func (m *middlewareDriver) Sign(ctx context.Context, id KeyID, digest []byte, mech Mechanism) ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	if len(digest) == 0 {
		return nil, ErrInvalidArgument
	}
	start := time.Now()
	var sig []byte
	var err error
	retryErr := m.withRetry(ctx, "Sign", func() error {
		sig, err = m.next.Sign(ctx, id, digest, mech)
		return err
	})
	m.log("Sign", id, err, time.Since(start))
	if retryErr != nil {
		return nil, err
	}
	return sig, err
}

func (m *middlewareDriver) Signer(ctx context.Context, id KeyID, mech Mechanism) (Signer, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	start := time.Now()
	s, err := m.next.Signer(ctx, id, mech)
	m.log("Signer", id, err, time.Since(start))
	return s, err
}

func (m *middlewareDriver) DeleteKey(ctx context.Context, id KeyID) error {
	if err := id.Validate(); err != nil {
		return err
	}
	start := time.Now()
	err := m.next.DeleteKey(ctx, id)
	m.log("DeleteKey", id, err, time.Since(start))
	return err
}

func (m *middlewareDriver) ListKeys(ctx context.Context) ([]KeyInfo, error) {
	start := time.Now()
	keys, err := m.next.ListKeys(ctx)
	m.log("ListKeys", KeyID{}, err, time.Since(start))
	return keys, err
}

func (m *middlewareDriver) Info(ctx context.Context) (*SlotInfo, error) {
	start := time.Now()
	info, err := m.next.Info(ctx)
	m.log("Info", KeyID{}, err, time.Since(start))
	return info, err
}

func (m *middlewareDriver) Close() error {
	start := time.Now()
	err := m.next.Close()
	m.log("Close", KeyID{}, err, time.Since(start))
	return err
}

var _ Driver = (*middlewareDriver)(nil)
