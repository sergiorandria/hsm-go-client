package hsm

import (
	"crypto/tls"
	"fmt"
	"time"

	hhttp "github.com/sergiorandria/hsm-go-client/hsm/http"
	"github.com/sergiorandria/hsm-go-client/hsm/pkcs11"
)

// Mechanism identifies the signing mechanism.
type Mechanism string

const (
	MechanismECDSASHA256    Mechanism = "ECDSA-SHA256"
	MechanismECDSASHA384    Mechanism = "ECDSA-SHA384"
	MechanismECDSASHA512    Mechanism = "ECDSA-SHA512"
	MechanismRSAPKCS1SHA256 Mechanism = "RSA-PKCS1-SHA256"
	MechanismRSAPSSSHA256   Mechanism = "RSA-PSS-SHA256"
	MechanismEd25519        Mechanism = "Ed25519"
)

// KeySpec describes a key to generate inside the HSM.
type KeySpec struct {
	Label       string    // CKA_LABEL, human readable, e.g. "alice"
	ID          []byte    // CKA_ID, unique bytes (if nil, derived from Label)
	Mechanism   Mechanism // signing mechanism; determines key type
	Curve       string    // for ECDSA: "P-256", "P-384", "P-521" (default P-256)
	Bits        int       // for RSA: 2048, 3072, 4096 (default 2048)
	Extractable bool      // false = private key never leaves HSM (recommended)
}

// KeyID identifies a key in the HSM.
type KeyID struct {
	Label string
	ID    []byte
}

// KeyInfo is returned from key operations.
type KeyInfo struct {
	ID           KeyID
	Algorithm    string // e.g. "ECDSA-P256", "RSA-2048"
	PublicKeyPEM string
}

// SlotInfo describes the HSM slot/token.
type SlotInfo struct {
	SlotID          uint
	SlotDescription string
	TokenLabel      string
	ManufacturerID  string
	Model           string
}

// Driver is defined in interfaces.go (composes KeyManager, Crypto, Lifecycle).

// DriverConfig selects the backend.
type DriverConfig struct {
	Backend string // "http" or "pkcs11"
	HTTP    HTTPConfig
	PKCS11  PKCS11Config
}

// HTTPConfig re-uses the HTTP microcontroller config.
type HTTPConfig struct {
	BaseURL        string
	BearerToken    string
	ChunkSize      int
	TimeoutSeconds int
	TLSConfig      *tls.Config `json:"-"` // mTLS: set RootCAs, GetClientCertificate, MinVersion
}

// PKCS11Config is for industrial HSMs via PKCS#11.
// Single USB/network-attached device: single slot label, PIN, one library path.
type PKCS11Config struct {
	LibraryPath string // e.g. "/usr/lib/softhsm/libsofthsm2.so" or "/usr/local/lib/libCryptoki2.so"
	SlotID      *uint  // optional: if nil, find by TokenLabel
	TokenLabel  string // e.g. "hsm-go-token"
	PIN         string // user PIN (CKU_USER); read from env, never log
	MaxSessions int    // default 4
}

func init() {
	// Register core backends via registry pattern (Factory). Vendor adapters self-register.
	RegisterBackend("http", func(cfg DriverConfig, opts ...Option) (Driver, error) {
		hCfg := hhttp.Config{
			BaseURL:     cfg.HTTP.BaseURL,
			BearerToken: cfg.HTTP.BearerToken,
			ChunkSize:   cfg.HTTP.ChunkSize,
			Timeout:     time.Duration(cfg.HTTP.TimeoutSeconds) * time.Second,
			TLSConfig:   cfg.HTTP.TLSConfig,
		}
		if hCfg.Timeout == 0 {
			hCfg.Timeout = 30 * time.Second
		}
		_ = applyOptions(opts...)
		return NewHTTPDriver(hhttp.NewClient(hCfg)), nil
	})
	RegisterBackend("microcontroller", func(cfg DriverConfig, opts ...Option) (Driver, error) {
		hCfg := hhttp.Config{
			BaseURL:     cfg.HTTP.BaseURL,
			BearerToken: cfg.HTTP.BearerToken,
			ChunkSize:   cfg.HTTP.ChunkSize,
			Timeout:     time.Duration(cfg.HTTP.TimeoutSeconds) * time.Second,
			TLSConfig:   cfg.HTTP.TLSConfig,
		}
		if hCfg.Timeout == 0 {
			hCfg.Timeout = 30 * time.Second
		}
		_ = applyOptions(opts...)
		return NewHTTPDriver(hhttp.NewClient(hCfg)), nil
	})
	RegisterBackend("mcu", func(cfg DriverConfig, opts ...Option) (Driver, error) {
		hCfg := hhttp.Config{
			BaseURL:     cfg.HTTP.BaseURL,
			BearerToken: cfg.HTTP.BearerToken,
			ChunkSize:   cfg.HTTP.ChunkSize,
			Timeout:     time.Duration(cfg.HTTP.TimeoutSeconds) * time.Second,
			TLSConfig:   cfg.HTTP.TLSConfig,
		}
		if hCfg.Timeout == 0 {
			hCfg.Timeout = 30 * time.Second
		}
		_ = applyOptions(opts...)
		return NewHTTPDriver(hhttp.NewClient(hCfg)), nil
	})
	RegisterBackend("esp32", func(cfg DriverConfig, opts ...Option) (Driver, error) {
		hCfg := hhttp.Config{
			BaseURL:     cfg.HTTP.BaseURL,
			BearerToken: cfg.HTTP.BearerToken,
			ChunkSize:   cfg.HTTP.ChunkSize,
			Timeout:     time.Duration(cfg.HTTP.TimeoutSeconds) * time.Second,
			TLSConfig:   cfg.HTTP.TLSConfig,
		}
		if hCfg.Timeout == 0 {
			hCfg.Timeout = 30 * time.Second
		}
		_ = applyOptions(opts...)
		return NewHTTPDriver(hhttp.NewClient(hCfg)), nil
	})
	RegisterBackend("pkcs11", func(cfg DriverConfig, opts ...Option) (Driver, error) {
		_ = applyOptions(opts...)
		pCfg := pkcs11.Config{
			LibraryPath: cfg.PKCS11.LibraryPath,
			SlotID:      cfg.PKCS11.SlotID,
			TokenLabel:  cfg.PKCS11.TokenLabel,
			PIN:         cfg.PKCS11.PIN,
			MaxSessions: cfg.PKCS11.MaxSessions,
		}
		return NewPKCS11Driver(pCfg)
	})
}

// NewDriver creates a Driver via backend registry. Supports Options for production (logger, retry, metrics).
// Validates config and wraps with middleware (logging/metrics/retry) for production.
func NewDriver(cfg DriverConfig, opts ...Option) (Driver, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	o := applyOptions(opts...)
	factory, ok := getFactory(cfg.Backend)
	if !ok {
		return nil, fmt.Errorf("%w: unknown backend %q (registered: %v)", ErrInvalidArgument, cfg.Backend, ListBackends())
	}
	d, err := factory(cfg, opts...)
	if err != nil {
		return nil, err
	}
	// Decorate with production middleware (logging/metrics/retry)
	return wrapWithMiddleware(d, o), nil
}
