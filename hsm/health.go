package hsm

import (
	"context"
	"fmt"
	"time"
)

// HealthStatus for production readiness/liveness probes.
type HealthStatus struct {
	Backend       string        `json:"backend"`
	Status        string        `json:"status"` // ok, degraded, down
	Latency       time.Duration `json:"latency"`
	SlotID        uint          `json:"slot_id,omitempty"`
	TokenLabel    string        `json:"token_label,omitempty"`
	ManufacturerID string       `json:"manufacturer_id,omitempty"`
	Error         string        `json:"error,omitempty"`
	CheckedAt     time.Time     `json:"checked_at"`
}

// HealthChecker extends Lifecycle for production probes. Driver already implements Info().
type HealthChecker interface {
	Health(ctx context.Context) (*HealthStatus, error)
}

// Health implements HealthChecker for any Driver via Info() + timing.
func Health(ctx context.Context, d Driver, backend string) (*HealthStatus, error) {
	start := time.Now()
	info, err := d.Info(ctx)
	lat := time.Since(start)
	status := &HealthStatus{
		Backend:   backend,
		CheckedAt: time.Now(),
		Latency:   lat,
	}
	if err != nil {
		status.Status = "down"
		status.Error = err.Error()
		return status, fmt.Errorf("%w: health check failed: %v", ErrDeviceNotReady, err)
	}
	status.Status = "ok"
	status.SlotID = info.SlotID
	status.TokenLabel = info.TokenLabel
	status.ManufacturerID = info.ManufacturerID
	return status, nil
}
