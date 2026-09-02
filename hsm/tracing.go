package hsm

import (
	"context"
)

// Tracing via context: production can plug OpenTelemetry by wrapping context.
// We avoid hard dependency on otel; provide helpers for span naming.

type ctxKey string

const spanKey ctxKey = "hsm.span"

// WithSpan stores span name in context for middleware to log/trace.
func WithSpan(ctx context.Context, op string) context.Context {
	return context.WithValue(ctx, spanKey, op)
}

func spanFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(spanKey).(string); ok {
		return v
	}
	return ""
}
