package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"
	"log"
	"os"

	"github.com/sergiorandria/hsm-go-client/hsm"
)

func main() {
	// Industrial HSM via PKCS#11 (Thales Luna, nShield, Utimaco, YubiHSM2, AWS CloudHSM, SoftHSM2)
	// Single USB/network-attached device.
	lib := os.Getenv("PKCS11_LIB")
	if lib == "" {
		lib = "/usr/lib/softhsm/libsofthsm2.so"
	}
	pin := os.Getenv("PKCS11_PIN")
	if pin == "" {
		pin = "1234"
	}
	tokenLabel := os.Getenv("PKCS11_TOKEN")
	if tokenLabel == "" {
		tokenLabel = "test-token"
	}
	cfg := hsm.DriverConfig{
		Backend: "pkcs11",
		PKCS11: hsm.PKCS11Config{
			LibraryPath: lib,
			TokenLabel:  tokenLabel,
			PIN:         pin,
			MaxSessions: 2,
		},
	}
	driver, err := hsm.NewDriver(cfg)
	if err != nil {
		log.Fatalf("Failed to create PKCS#11 driver (CGO_ENABLED=1 required): %v", err)
	}
	defer driver.Close()

	ctx := context.Background()
	info, err := driver.Info(ctx)
	if err != nil {
		log.Fatalf("Info failed: %v", err)
	}
	fmt.Printf("Connected to %s (%s) slot %d\n", info.TokenLabel, info.ManufacturerID, info.SlotID)

	// 1. Generate key inside HSM (private never leaves device)
	spec := hsm.KeySpec{Label: "alice", Mechanism: hsm.MechanismECDSASHA256, Curve: "P-256"}
	_ = driver.DeleteKey(ctx, hsm.KeyID{Label: "alice"}) // clean for demo
	ki, err := driver.GenerateKey(ctx, spec)
	if err != nil {
		log.Fatalf("GenerateKey: %v", err)
	}
	fmt.Printf("✓ Key created: %s\n%s\n", ki.Algorithm, ki.PublicKeyPEM)

	// 2. Host hashes file, HSM signs digest (industrial flow)
	data := []byte("This is the document to sign")
	digest := sha256.Sum256(data)
	fmt.Printf("Digest SHA256: %x\n", digest)

	sig, err := driver.Sign(ctx, hsm.KeyID{Label: "alice"}, digest[:], hsm.MechanismECDSASHA256)
	if err != nil {
		log.Fatalf("Sign: %v", err)
	}
	fmt.Printf("Signature ASN.1 DER len %d\n", len(sig))

	// 3. Verify with Go stdlib (or use driver.Signer as crypto.Signer)
	pub, _, err := driver.GetPublicKey(ctx, hsm.KeyID{Label: "alice"})
	if err != nil {
		log.Fatalf("GetPublicKey: %v", err)
	}
	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		log.Fatalf("unexpected pub type %T", pub)
	}
	if ecdsa.VerifyASN1(ecdsaPub, digest[:], sig) {
		fmt.Println("✓ Signature verified")
	} else {
		log.Fatal("verification failed")
	}

	// 4. Use as crypto.Signer for tls/x509
	signer, err := driver.Signer(ctx, hsm.KeyID{Label: "alice"}, hsm.MechanismECDSASHA256)
	if err != nil {
		log.Fatalf("Signer: %v", err)
	}
	fmt.Printf("Signer ready: %T, public %T\n", signer, signer.Public())

	// Also works via microcontroller HTTP backend:
	// httpClient := hsm.NewHTTPDriver(httpClient) or hsm.NewDriver(DriverConfig{Backend: "http", HTTP: ...})
}
