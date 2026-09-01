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

// Config for AWS CloudHSM via libcloudhsm_pkcs11.so.
type Config struct {
	LibraryPath string // e.g. /opt/cloudhsm/lib/libcloudhsm_pkcs11.so
	TokenLabel  string // e.g. "cavium"
	PIN         string // CU "username:password"
	SlotID      *uint
	MaxSessions int // default 4
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
	// CloudHSM enforces CKA_TOKEN=true and cluster sync; generic already sets it.
	// Adapter retries ListKeys until key visible (cluster async).
	ki, err := a.inner.GenerateKey(ctx, spec)
	if err != nil {
		return nil, err
	}
	// Retry until visible on cluster (up to 2s)
	for i := 0; i < 5; i++ {
		if _, _, err := a.inner.GetPublicKey(ctx, ki.ID); err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	return ki, nil
}

func (a *adapter) GetPublicKey(ctx context.Context, id hsm.KeyID) (crypto.PublicKey, string, error) {
	return a.inner.GetPublicKey(ctx, id)
}

func (a *adapter) Sign(ctx context.Context, id hsm.KeyID, digest []byte, mech hsm.Mechanism) ([]byte, error) {
	// CloudHSM same as generic for CKM_ECDSA; no translation needed.
	return a.inner.Sign(ctx, id, digest, mech)
}

func (a *adapter) Signer(ctx context.Context, id hsm.KeyID, mech hsm.Mechanism) (hsm.Signer, error) {
	return a.inner.Signer(ctx, id, mech)
}

func (a *adapter) DeleteKey(ctx context.Context, id hsm.KeyID) error {
	return a.inner.DeleteKey(ctx, id)
}

func (a *adapter) ListKeys(ctx context.Context) ([]hsm.KeyInfo, error) {
	return a.inner.ListKeys(ctx)
}

func (a *adapter) Info(ctx context.Context) (*hsm.SlotInfo, error) {
	return a.inner.Info(ctx)
}

func (a *adapter) Close() error { return a.inner.Close() }

var _ hsm.Driver = (*adapter)(nil)
