//go:build cgo

package cloudhsm

import (
	"context"
	"crypto"
	"fmt"
	"strings"
	"time"

	"github.com/sergiorandria/hsm-go-client/hsm"
	"github.com/sergiorandria/hsm-go-client/hsm/pkcs11"
)

func init() {
	hsm.RegisterBackend("cloudhsm", func(cfg hsm.DriverConfig, opts ...hsm.Option) (hsm.Driver, error) {
		cCfg := Config{
			LibraryPath: cfg.PKCS11.LibraryPath,
			TokenLabel:  cfg.PKCS11.TokenLabel,
			PIN:         cfg.PKCS11.PIN,
			SlotID:      cfg.PKCS11.SlotID,
			MaxSessions: cfg.PKCS11.MaxSessions,
		}
		return NewDriver(cCfg)
	})
	hsm.RegisterBackend("aws-cloudhsm", func(cfg hsm.DriverConfig, opts ...hsm.Option) (hsm.Driver, error) {
		cCfg := Config{
			LibraryPath: cfg.PKCS11.LibraryPath,
			TokenLabel:  cfg.PKCS11.TokenLabel,
			PIN:         cfg.PKCS11.PIN,
			SlotID:      cfg.PKCS11.SlotID,
			MaxSessions: cfg.PKCS11.MaxSessions,
		}
		return NewDriver(cCfg)
	})
}

// Config for AWS CloudHSM via libcloudhsm_pkcs11.so. HA cluster philosophy.
type Config struct {
	LibraryPath string // e.g. /opt/cloudhsm/lib/libcloudhsm_pkcs11.so
	TokenLabel  string // e.g. "cavium"
	PIN         string // CU "username:password"
	SlotID      *uint
	MaxSessions int    // default 4 (HA cluster needs >1 for failover)
	ClusterID   string // optional cluster identifier for health check
}

type adapter struct {
	inner hsm.Driver
	cfg   Config
}

func NewDriver(cfg Config) (hsm.Driver, error) {
	if cfg.LibraryPath == "" {
		cfg.LibraryPath = "/opt/cloudhsm/lib/libcloudhsm_pkcs11.so"
	}
	if cfg.MaxSessions == 0 {
		cfg.MaxSessions = 4
	}
	pin := normalizePIN(cfg.PIN)
	inner, err := hsm.NewPKCS11Driver(pkcs11.Config{
		LibraryPath: cfg.LibraryPath,
		TokenLabel:  cfg.TokenLabel,
		PIN:         pin,
		SlotID:      cfg.SlotID,
		MaxSessions: cfg.MaxSessions,
	})
	if err != nil {
		return nil, fmt.Errorf("cloudhsm: %w (ensure cloudhsm_client running and cluster configured)", err)
	}
	return &adapter{inner: inner, cfg: cfg}, nil
}

func normalizePIN(pin string) string {
	// CloudHSM expects "username:password" for CU. Generic assumes plain PIN.
	// If no colon, keep as-is (some tools use "password" with default CU).
	if pin == "" {
		return pin
	}
	if strings.Contains(pin, ":") {
		return pin
	}
	return pin
}

func (a *adapter) GenerateKey(ctx context.Context, spec hsm.KeySpec) (*hsm.KeyInfo, error) {
	// CloudHSM HA: cluster sync async, key must be visible on all HSMs. Retry with backoff.
	ki, err := a.inner.GenerateKey(ctx, spec)
	if err != nil {
		return nil, mapDeviceError(err)
	}
	// Retry until visible on cluster (up to 3s, HA)
	for i := 0; i < 6; i++ {
		if _, _, err := a.inner.GetPublicKey(ctx, ki.ID); err == nil {
			return ki, nil
		} else if isTransientClusterError(err) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(100*(i+1)) * time.Millisecond):
			}
			continue
		} else {
			break
		}
	}
	return ki, nil
}

func (a *adapter) GetPublicKey(ctx context.Context, id hsm.KeyID) (crypto.PublicKey, string, error) {
	pub, pem, err := a.inner.GetPublicKey(ctx, id)
	if err != nil {
		return nil, "", mapDeviceError(err)
	}
	return pub, pem, nil
}

func (a *adapter) Sign(ctx context.Context, id hsm.KeyID, digest []byte, mech hsm.Mechanism) ([]byte, error) {
	// CloudHSM HA: retry on CKR_DEVICE_REMOVED / token not present (device failover)
	var sig []byte
	var err error
	for i := 0; i < 3; i++ {
		sig, err = a.inner.Sign(ctx, id, digest, mech)
		if err == nil {
			return sig, nil
		}
		if !isTransientClusterError(err) || i == 2 {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(100*(i+1)) * time.Millisecond):
		}
	}
	return nil, mapDeviceError(err)
}

func (a *adapter) Signer(ctx context.Context, id hsm.KeyID, mech hsm.Mechanism) (hsm.Signer, error) {
	return a.inner.Signer(ctx, id, mech)
}

func (a *adapter) DeleteKey(ctx context.Context, id hsm.KeyID) error {
	return a.inner.DeleteKey(ctx, id)
}

func (a *adapter) ListKeys(ctx context.Context) ([]hsm.KeyInfo, error) {
	keys, err := a.inner.ListKeys(ctx)
	if err != nil {
		return nil, mapDeviceError(err)
	}
	return keys, nil
}

func (a *adapter) Info(ctx context.Context) (*hsm.SlotInfo, error) {
	// CloudHSM Info also serves as cluster health check; surface ManufacturerID "Cavium"
	info, err := a.inner.Info(ctx)
	if err != nil {
		return nil, mapDeviceError(err)
	}
	return info, nil
}

// mapDeviceError converts PKCS#11 transient cluster errors to ErrDeviceNotReady for middleware retry.
func mapDeviceError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	// CloudHSM cluster errors: CKR_DEVICE_REMOVED (0x30), CKR_TOKEN_NOT_PRESENT (0xE0), CKR_DEVICE_ERROR
	if contains(msg, "0x30") || contains(msg, "0xE0") || contains(msg, "CKR_DEVICE_REMOVED") || contains(msg, "CKR_TOKEN_NOT_PRESENT") || contains(msg, "CKR_DEVICE_ERROR") {
		return fmt.Errorf("%w: %v", hsm.ErrDeviceNotReady, err)
	}
	return err
}

func isTransientClusterError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "0x30") || contains(msg, "0xE0") || contains(msg, "CKR_DEVICE_REMOVED") || contains(msg, "CKR_TOKEN_NOT_PRESENT")
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func (a *adapter) Close() error { return a.inner.Close() }

var _ hsm.Driver = (*adapter)(nil)
