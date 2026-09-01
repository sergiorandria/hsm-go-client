// Package yubihsm provides a YubiHSM2 adapter for the generic hsm.Driver.
//
// YubiHSM2 is a USB HSM with two access paths: native yubihsm-connector HTTP
// and PKCS#11 via yubihsm_pkcs11.so. This adapter uses PKCS#11 but adapts
// YubiHSM quirks without modifying the generic hsm/pkcs11 driver:
//
//   - PIN format: authKey ID 0x0001 + password (e.g. "0001:password") vs generic "1234"
//   - Slot selection via TokenLabel (YubiHSM connector exposes label "YubiHSM")
//   - ECDSA mechanism: YubiHSM supports CKM_ECDSA for raw digest (same as generic)
//   - Ed25519: vendor supports CKM_EDDSA, generic stub will be adapted here
//   - Library path: /usr/lib/x86_64-linux-gnu/pkcs11/yubihsm_pkcs11.so
//   - Requires yubihsm-connector daemon running: yubihsm-connector -d
//
// Contradiction handled: generic PKCS#11 assumes PIN is CKA_USER PIN; YubiHSM
// requires "authKeyID:password" and may need connector URL. Adapter translates.
//
// Usage:
//
//	import "github.com/sergiorandria/hsm-go-client/hsm/yubihsm"
//	d, err := yubihsm.NewDriver(yubihsm.Config{
//	  LibraryPath: "/usr/lib/x86_64-linux-gnu/pkcs11/yubihsm_pkcs11.so",
//	  TokenLabel:  "YubiHSM",
//	  PIN:         "0001:password",
//	  ConnectorURL: "http://127.0.0.1:12345",
//	})
package yubihsm
