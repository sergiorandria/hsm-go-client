package hsm

import (
	"sync"
	"time"
)

// MetricsRecorder records operation metrics for production observability.
// Implement with Prometheus, OpenTelemetry, or custom.
type MetricsRecorder interface {
	Record(op string, duration time.Duration, err error, labels map[string]string)
}

// InMemoryMetrics is a simple in-memory recorder for testing or fallback.
type InMemoryMetrics struct {
	mu      sync.Mutex
	records []MetricRecord
}

type MetricRecord struct {
	Op       string
	Duration time.Duration
	Err      error
	Labels   map[string]string
}

func (m *InMemoryMetrics) Record(op string, duration time.Duration, err error, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, MetricRecord{Op: op, Duration: duration, Err: err, Labels: labels})
}

func (m *InMemoryMetrics) Records() []MetricRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]MetricRecord, len(m.records))
	copy(out, m.records)
	return out
}

// Prometheus-style hook adapter: use WithMetrics to bridge to your registry.
func MetricsHookFromRecorder(rec MetricsRecorder) func(op string, duration time.Duration, err error) {
	return func(op string, duration time.Duration, err error) {
		labels := map[string]string{"backend": "unknown"}
		if err != nil {
			labels["status"] = "error"
		} else {
			labels["status"] = "ok"
		}
		rec.Record(op, duration, err, labels)
	}
}
