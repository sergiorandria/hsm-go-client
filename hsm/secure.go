package hsm

import (
	"os"
	"strings"
)

// SecureString holds sensitive data (PIN, token) and avoids logging.
// For production: load from env/file, zero after use where possible.
type SecureString string

func (s SecureString) String() string { return "****" }
func (s SecureString) Raw() string    { return string(s) }

// PINFromEnv loads PIN from HSM_PIN or HSM_PKCS11_PIN env, falls back to value.
// Never log the raw value; use SecureString.
func PINFromEnv(envKeys ...string) SecureString {
	keys := append(envKeys, "HSM_PIN", "HSM_PKCS11_PIN", "PKCS11_PIN")
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return SecureString(v)
		}
	}
	return ""
}

// BearerTokenFromEnv loads token from env securely.
func BearerTokenFromEnv(envKeys ...string) SecureString {
	keys := append(envKeys, "HSM_BEARER_TOKEN", "HSM_TOKEN", "BEARER_TOKEN")
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return SecureString(v)
		}
	}
	return ""
}

// Redact returns redacted string for logs (never expose secrets).
func Redact(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(s)-4) + s[len(s)-4:]
}
