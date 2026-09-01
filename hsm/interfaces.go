package hsm

import (
	"context"
	"crypto"
)

// KeyManager manages lifecycle of keys inside the HSM (ISP: key ops separated from crypto).
type KeyManager interface {
	GenerateKey(ctx context.Context, spec KeySpec) (*KeyInfo, error)
	GetPublicKey(ctx context.Context, id KeyID) (crypto.PublicKey, string, error)
	DeleteKey(ctx context.Context, id KeyID) error
	ListKeys(ctx context.Context) ([]KeyInfo, error)
}

// Crypto performs signing with keys that never leave the HSM.
type Crypto interface {
	Sign(ctx context.Context, id KeyID, digest []byte, mech Mechanism) ([]byte, error)
	Signer(ctx context.Context, id KeyID, mech Mechanism) (Signer, error)
}

// Lifecycle manages observability and teardown.
type Lifecycle interface {
	Info(ctx context.Context) (*SlotInfo, error)
	Close() error
}

// Signer is a crypto.Signer bound to a specific key inside the HSM.
type Signer interface {
	crypto.Signer
	KeyID() KeyID
}

// Driver composes KeyManager, Crypto, Lifecycle for backward compat.
// Prefer depending on the narrow interface (e.g. KeyManager) in callers.
type Driver interface {
	KeyManager
	Crypto
	Lifecycle
}

// Ensure http and pkcs11 drivers satisfy segregated interfaces.
var (
	_ KeyManager = (Driver)(nil)
	_ Crypto     = (Driver)(nil)
	_ Lifecycle  = (Driver)(nil)
)
