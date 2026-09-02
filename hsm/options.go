package hsm

import (
	"log/slog"
	"time"
)

// Option configures a driver via functional options. Scalable, no breaking config changes.
type Option func(*options)

type options struct {
	logger      *slog.Logger
	timeout     time.Duration
	maxRetries  int
	retryDelay  time.Duration
	metricsHook func(op string, duration time.Duration, err error)
}

// WithLogger sets structured logger for all driver ops (default: slog.Default).
func WithLogger(l *slog.Logger) Option {
	return func(o *options) { o.logger = l }
}

// WithTimeout sets per-op timeout (overrides HTTPConfig.Timeout / PKCS#11 session wait).
func WithTimeout(d time.Duration) Option {
	return func(o *options) { o.timeout = d }
}

// WithRetry sets retry for transient errors (DeviceNotReady, Timeout). Default 0.
func WithRetry(maxRetries int, delay time.Duration) Option {
	return func(o *options) {
		o.maxRetries = maxRetries
		o.retryDelay = delay
	}
}

// WithMetrics hooks op latency/error for Prometheus/OpenTelemetry.
func WithMetrics(hook func(op string, duration time.Duration, err error)) Option {
	return func(o *options) { o.metricsHook = hook }
}

func applyOptions(opts ...Option) *options {
	o := &options{
		timeout:    30 * time.Second,
		maxRetries: 0,
	}
	for _, fn := range opts {
		fn(o)
	}
	return o
}
