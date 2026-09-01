//go:build cgo

package pkcs11

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDriverMissingLibrary(t *testing.T) {
	_, err := NewDriver(Config{LibraryPath: ""})
	assert.Error(t, err)
	_, err = NewDriver(Config{LibraryPath: "/nonexistent.so"})
	assert.Error(t, err)
}

func TestMechanismFor(t *testing.T) {
	m, err := mechanismFor(MechanismECDSASHA256)
	require.NoError(t, err)
	require.NotEmpty(t, m)
	_, err = mechanismFor("unknown")
	assert.Error(t, err)
}

func TestPKCS11IntegrationWithSoftHSM(t *testing.T) {
	lib := os.Getenv("PKCS11_LIB")
	if lib == "" {
		// try common paths
		for _, p := range []string{"/usr/lib/softhsm/libsofthsm2.so", "/usr/lib/x86_64-linux-gnu/softhsm/libsofthsm2.so", "/usr/local/lib/softhsm/libsofthsm2.so"} {
			if _, err := os.Stat(p); err == nil {
				lib = p
				break
			}
		}
	}
	if lib == "" {
		t.Skip("PKCS11_LIB not set and no default softhsm found, skipping integration test")
	}
	pin := os.Getenv("PKCS11_PIN")
	if pin == "" {
		pin = "1234"
	}
	tokenLabel := os.Getenv("PKCS11_TOKEN")
	if tokenLabel == "" {
		tokenLabel = "test-token"
	}
	// SoftHSM needs initialization; assume token already initialized via softhsm2-util
	// If not, try to proceed and error will show.
	cfg := Config{
		LibraryPath: lib,
		TokenLabel:  tokenLabel,
		PIN:         pin,
		MaxSessions: 2,
	}
	driver, err := NewDriver(cfg)
	if err != nil {
		t.Skipf("SoftHSM not available: %v", err)
	}
	defer driver.Close()

	ctx := context.Background()
	info, err := driver.Info(ctx)
	require.NoError(t, err)
	t.Logf("Slot %d Token %s", info.SlotID, info.TokenLabel)

	spec := KeySpec{Label: "test-ecdsa-p256", Mechanism: MechanismECDSASHA256, Curve: "P-256"}
	// Clean up any previous key
	_ = driver.DeleteKey(ctx, KeyID{Label: spec.Label})
	ki, err := driver.GenerateKey(ctx, spec)
	require.NoError(t, err)
	assert.NotEmpty(t, ki.PublicKeyPEM)
	assert.Contains(t, ki.PublicKeyPEM, "BEGIN PUBLIC KEY")
	t.Logf("Generated %s", ki.Algorithm)

	pub, pemStr, err := driver.GetPublicKey(ctx, KeyID{Label: spec.Label})
	require.NoError(t, err)
	assert.NotEmpty(t, pemStr)
	assert.NotNil(t, pub)
	_, ok := pub.(*ecdsa.PublicKey)
	assert.True(t, ok)

	digest := sha256.Sum256([]byte("hello hsm"))
	sig, err := driver.Sign(ctx, KeyID{Label: spec.Label}, digest[:], MechanismECDSASHA256)
	require.NoError(t, err)
	assert.NotEmpty(t, sig)
	// Verify with Go stdlib
	ecdsaPub := pub.(*ecdsa.PublicKey)
	assert.True(t, ecdsa.VerifyASN1(ecdsaPub, digest[:], sig))

	signer, err := driver.Signer(ctx, KeyID{Label: spec.Label}, MechanismECDSASHA256)
	require.NoError(t, err)
	sig2, err := signer.Sign(nil, digest[:], nil)
	require.NoError(t, err)
	assert.True(t, ecdsa.VerifyASN1(ecdsaPub, digest[:], sig2))

	list, err := driver.ListKeys(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 1)

	require.NoError(t, driver.DeleteKey(ctx, KeyID{Label: spec.Label}))
}
