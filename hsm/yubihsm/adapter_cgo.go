//go:build cgo

package yubihsm

import (
	"context"
	"crypto"
	"fmt"
	"strings"

	"github.com/sergiorandria/hsm-go-client/hsm"
	"github.com/sergiorandria/hsm-go-client/hsm/pkcs11"
)

// Config for YubiHSM2 via PKCS#11.
// Follows Yubico docs: yubihsm_pkcs11.so + yubihsm-connector.
type Config struct {
	LibraryPath  string // e.g. /usr/lib/x86_64-linux-gnu/pkcs11/yubihsm_pkcs11.so
	TokenLabel   string // e.g. "YubiHSM" (default)
	PIN          string // "authKeyID:password", e.g. "0001:password" or just "password" (adapter normalizes)
	SlotID       *uint
	MaxSessions  int    // default 2 (USB single device)
	ConnectorURL string // e.g. http://127.0.0.1:12345 for YUBIHSM_PKCS11_CONF
}

type adapter struct {
	inner hsm.Driver
	cfg   Config
}

// NewDriver creates a hsm.Driver for YubiHSM2 without modifying generic pkcs11 driver.
// Adapter handles PIN normalization and connector env.
func NewDriver(cfg Config) (hsm.Driver, error) {
	if cfg.LibraryPath == "" {
		cfg.LibraryPath = "/usr/lib/x86_64-linux-gnu/pkcs11/yubihsm_pkcs11.so"
	}
	if cfg.TokenLabel == "" {
		cfg.TokenLabel = "YubiHSM"
	}
	if cfg.MaxSessions == 0 {
		cfg.MaxSessions = 2
	}
	pin := normalizePIN(cfg.PIN)
	// YubiHSM connector URL via env YUBIHSM_PKCS11_CONF or ConnectorURL handled by library config file.
	// Adapter sets env if provided (not modifying global state beyond this process's env).
	if cfg.ConnectorURL != "" {
		// yubihsm_pkcs11 reads connector from config; we document to set YUBIHSM_PKCS11_CONF.
		// Here we just note; actual file must be created by user. No env change to avoid side-effects.
		_ = cfg.ConnectorURL
	}
	inner, err := hsm.NewPKCS11Driver(pkcs11.Config{
		LibraryPath: cfg.LibraryPath,
		TokenLabel:  cfg.TokenLabel,
		PIN:         pin,
		SlotID:      cfg.SlotID,
		MaxSessions: cfg.MaxSessions,
	})
	if err != nil {
		return nil, fmt.Errorf("yubihsm: %w", err)
	}
	return &adapter{inner: inner, cfg: cfg}, nil
}

func normalizePIN(pin string) string {
	// YubiHSM expects "0001:password" if authKey ID not given.
	// Generic assumes plain password; adapt.
	if pin == "" {
		return pin
	}
	if strings.Contains(pin, ":") {
		return pin
	}
	// Assume default authKey 0x0001
	return "0001:" + pin
}

func (a *adapter) GenerateKey(ctx context.Context, spec hsm.KeySpec) (*hsm.KeyInfo, error) {
	// YubiHSM2 philosophy: capability-based, session-limited (2), AuthKey 0x0001.
	// Supports P-256/P-384 and Ed25519 (not P-521 via PKCS#11). Generic now handles Ed25519 via
	// CKM_EC_EDWARDS_KEY_PAIR_GEN + OID 1.3.101.112 + CKM_EDDSA (incremental fix pkcs11/driver_cgo.go:185).
	if spec.Mechanism == hsm.MechanismEd25519 {
		// Normalize Curve for Edwards; YubiHSM expects Ed25519 OID, not P-256
		if spec.Curve == "" {
			spec.Curve = "Ed25519"
		}
		// YubiHSM session: ensure capability sign-eddsa; generic handles via PKCS#11.
	}
	return a.inner.GenerateKey(ctx, spec)
}

func (a *adapter) GetPublicKey(ctx context.Context, id hsm.KeyID) (crypto.PublicKey, string, error) {
	return a.inner.GetPublicKey(ctx, id)
}

func (a *adapter) Sign(ctx context.Context, id hsm.KeyID, digest []byte, mech hsm.Mechanism) ([]byte, error) {
	// YubiHSM uses CKM_ECDSA for raw digest same as generic - no translation needed.
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
