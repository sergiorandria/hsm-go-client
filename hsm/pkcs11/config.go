//go:build cgo

package pkcs11

// Config for PKCS#11 single USB/network-attached device.
type Config struct {
	LibraryPath string // e.g. "/usr/lib/softhsm/libsofthsm2.so"
	SlotID      *uint  // optional: if nil, find by TokenLabel
	TokenLabel  string // e.g. "test-token"
	PIN         string // user PIN (CKU_USER)
	MaxSessions int    // default 4
	SO_PIN      string // optional SO pin for token init (SoftHSM)
}
