//go:build cgo

package hsm

import (
	"context"
	"crypto"
	"fmt"
	"io"

	"github.com/sergiorandria/hsm-go-client/hsm/pkcs11"
)

type pkcs11Driver struct {
	inner *pkcs11.Driver
}

func NewPKCS11Driver(cfg pkcs11.Config) (Driver, error) {
	d, err := pkcs11.NewDriver(cfg)
	if err != nil {
		return nil, err
	}
	return &pkcs11Driver{inner: d}, nil
}

func (d *pkcs11Driver) GenerateKey(ctx context.Context, spec KeySpec) (*KeyInfo, error) {
	innerSpec := pkcs11.KeySpec{
		Label:       spec.Label,
		ID:          spec.ID,
		Mechanism:   pkcs11.Mechanism(spec.Mechanism),
		Curve:       spec.Curve,
		Bits:        spec.Bits,
		Extractable: spec.Extractable,
	}
	ki, err := d.inner.GenerateKey(ctx, innerSpec)
	if err != nil {
		return nil, err
	}
	return &KeyInfo{ID: KeyID{Label: ki.ID.Label, ID: ki.ID.ID}, Algorithm: ki.Algorithm, PublicKeyPEM: ki.PublicKeyPEM}, nil
}

func (d *pkcs11Driver) GetPublicKey(ctx context.Context, id KeyID) (crypto.PublicKey, string, error) {
	return d.inner.GetPublicKey(ctx, pkcs11.KeyID{Label: id.Label, ID: id.ID})
}

func (d *pkcs11Driver) Sign(ctx context.Context, id KeyID, digest []byte, mech Mechanism) ([]byte, error) {
	return d.inner.Sign(ctx, pkcs11.KeyID{Label: id.Label, ID: id.ID}, digest, pkcs11.Mechanism(mech))
}

type pkcs11SignerWrapper struct {
	inner pkcs11.Signer
	keyID KeyID
}

func (s *pkcs11SignerWrapper) Public() crypto.PublicKey { return s.inner.Public() }
func (s *pkcs11SignerWrapper) KeyID() KeyID             { return s.keyID }
func (s *pkcs11SignerWrapper) Sign(r io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return s.inner.Sign(r, digest, opts)
}

func (d *pkcs11Driver) Signer(ctx context.Context, id KeyID, mech Mechanism) (Signer, error) {
	s, err := d.inner.Signer(ctx, pkcs11.KeyID{Label: id.Label, ID: id.ID}, pkcs11.Mechanism(mech))
	if err != nil {
		return nil, err
	}
	return &pkcs11SignerWrapper{inner: s, keyID: id}, nil
}

func (d *pkcs11Driver) DeleteKey(ctx context.Context, id KeyID) error {
	return d.inner.DeleteKey(ctx, pkcs11.KeyID{Label: id.Label, ID: id.ID})
}

func (d *pkcs11Driver) ListKeys(ctx context.Context) ([]KeyInfo, error) {
	list, err := d.inner.ListKeys(ctx)
	if err != nil {
		return nil, err
	}
	var out []KeyInfo
	for _, ki := range list {
		out = append(out, KeyInfo{ID: KeyID{Label: ki.ID.Label, ID: ki.ID.ID}, Algorithm: ki.Algorithm, PublicKeyPEM: ki.PublicKeyPEM})
	}
	return out, nil
}

func (d *pkcs11Driver) Info(ctx context.Context) (*SlotInfo, error) {
	si, err := d.inner.Info(ctx)
	if err != nil {
		return nil, err
	}
	return &SlotInfo{SlotID: si.SlotID, SlotDescription: si.SlotDescription, TokenLabel: si.TokenLabel, ManufacturerID: si.ManufacturerID, Model: si.Model}, nil
}

func (d *pkcs11Driver) Close() error { return d.inner.Close() }

// Update NewDriver factory to support both backends
func init() {
	_ = fmt.Sprintf
}
