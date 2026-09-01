// Package pkcs11 implements an industrial HSM driver via PKCS#11.
//
// Supports Thales Luna, Entrust nShield, Utimaco SecurityServer, YubiHSM2 (via pkcs11),
// AWS CloudHSM, and SoftHSM2 for testing. Requires CGO_ENABLED=1 and a PKCS#11 library.
//
// Single USB/network-attached device: one slot, one token label, PIN authentication.
//
// Example:
//
//	cfg := pkcs11.Config{
//	  LibraryPath: "/usr/lib/softhsm/libsofthsm2.so",
//	  TokenLabel:  "test-token",
//	  PIN:         "1234",
//	}
//	driver, err := pkcs11.NewDriver(cfg)
//	defer driver.Close()
//
//	spec := pkcs11.KeySpec{Label: "alice", Mechanism: pkcs11.MechanismECDSASHA256, Curve: "P-256"}
//	ki, _ := driver.GenerateKey(ctx, spec)
//	sig, _ := driver.Sign(ctx, pkcs11.KeyID{Label: "alice"}, digest, pkcs11.MechanismECDSASHA256)
package pkcs11
