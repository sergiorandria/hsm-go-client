package hsm

import (
	"bytes"
	"context"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"

	hhttp "github.com/sergiorandria/hsm-go-client/hsm/http"
)

// httpDriver wraps the generic microcontroller HTTP client to implement Driver.
type httpDriver struct {
	client *hhttp.Client
}

func NewHTTPDriver(client *hhttp.Client) Driver {
	return &httpDriver{client: client}
}

// NewHTTPDriverFromConfig convenience.
func NewHTTPDriverFromConfig(cfg hhttp.Config) Driver {
	return &httpDriver{client: hhttp.NewClient(cfg)}
}

func (d *httpDriver) GenerateKey(ctx context.Context, spec KeySpec) (*KeyInfo, error) {
	if spec.Label == "" {
		return nil, fmt.Errorf("Label required (maps to userId)")
	}
	resp, err := d.client.CreateKey(ctx, spec.Label, true)
	if err != nil {
		return nil, err
	}
	id := spec.ID
	if len(id) == 0 {
		id = []byte(spec.Label)
	}
	return &KeyInfo{
		ID:           KeyID{Label: spec.Label, ID: id},
		Algorithm:    resp.PublicKeyAlgorithm,
		PublicKeyPEM: resp.PublicKeyPEM,
	}, nil
}

func (d *httpDriver) GetPublicKey(ctx context.Context, id KeyID) (crypto.PublicKey, string, error) {
	return nil, "", fmt.Errorf("http driver: GetPublicKey not supported without prior GenerateKey; microcontroller has no key-fetch API (use GenerateKey response)")
}

func (d *httpDriver) Sign(ctx context.Context, id KeyID, digest []byte, mech Mechanism) ([]byte, error) {
	if len(digest) == 0 {
		return nil, fmt.Errorf("digest empty")
	}
	if id.Label == "" {
		return nil, fmt.Errorf("KeyID.Label required for http driver")
	}
	filename := fmt.Sprintf("digest-%s.bin", hex.EncodeToString(digest[:8]))
	if err := d.client.UploadFile(ctx, filename, bytes.NewReader(digest)); err != nil {
		return nil, fmt.Errorf("http driver upload digest: %w", err)
	}
	resp, err := d.client.Sign(ctx, id.Label, filename, nil)
	if err != nil {
		return nil, err
	}
	sig, err := base64.StdEncoding.DecodeString(resp.SignatureBase64)
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}
	return sig, nil
}

type httpSigner struct {
	driver *httpDriver
	keyID  KeyID
	mech   Mechanism
	pub    crypto.PublicKey
}

func (s *httpSigner) Public() crypto.PublicKey { return s.pub }
func (s *httpSigner) KeyID() KeyID             { return s.keyID }
func (s *httpSigner) Sign(_ io.Reader, digest []byte, _ crypto.SignerOpts) ([]byte, error) {
	return s.driver.Sign(context.Background(), s.keyID, digest, s.mech)
}

func (d *httpDriver) Signer(ctx context.Context, id KeyID, mech Mechanism) (Signer, error) {
	// Try to create/get pub via GenerateKey cache? For demo generate if not exists.
	ki, err := d.GenerateKey(ctx, KeySpec{Label: id.Label, ID: id.ID, Mechanism: mech})
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode([]byte(ki.PublicKeyPEM))
	if block == nil {
		return &httpSigner{driver: d, keyID: id, mech: mech, pub: nil}, nil
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return &httpSigner{driver: d, keyID: id, mech: mech, pub: nil}, nil
	}
	return &httpSigner{driver: d, keyID: id, mech: mech, pub: pub}, nil
}

func (d *httpDriver) DeleteKey(_ context.Context, _ KeyID) error {
	return fmt.Errorf("http driver: DeleteKey not supported on microcontroller prototype")
}

func (d *httpDriver) ListKeys(_ context.Context) ([]KeyInfo, error) {
	return nil, fmt.Errorf("http driver: ListKeys not supported")
}

func (d *httpDriver) Info(_ context.Context) (*SlotInfo, error) {
	return &SlotInfo{
		SlotDescription: "HTTP Microcontroller HSM",
		TokenLabel:      "http-mcu",
		ManufacturerID:  "self-made",
		Model:           "ESP32/Raspberry Pi",
	}, nil
}

func (d *httpDriver) Close() error { return nil }
