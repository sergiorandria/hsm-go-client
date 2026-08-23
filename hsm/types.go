package hsm

import "encoding/base64"

// Signature represents a signature returned from the HSM.
type Signature struct {
	Algorithm string // e.g., "ECDSA-SHA256"
	Base64    string // Base64-encoded signature
}

// Bytes returns the signature as bytes (decoded from base64).
func (s *Signature) Bytes() ([]byte, error) {
	return base64.StdEncoding.DecodeString(s.Base64)
}

// Hash represents a hash computed by the HSM.
type Hash struct {
	Algorithm string // e.g., "SHA-256"
	Hex       string // Hexadecimal representation
}

// Key represents a cryptographic key.
type Key struct {
	UserID       string // User identifier
	Algorithm    string // e.g., "ECDSA-P256"
	PublicKeyPEM string // Public key in PEM format
	CreatedAt    string // Creation timestamp (optional)
	ExpiresAt    string // Expiration timestamp (optional)
}

// SignatureBundle bundles the result of a signing operation.
type SignatureBundle struct {
	UserID    string
	Filename  string
	Hash      Hash
	Signature Signature
	SignedAt  string // Optional timestamp from HSM
	Metadata  interface{}
}

// ErrorResponse represents an error response from the HSM.
type ErrorResponse struct {
	Status  string `json:"status"`
	Error   string `json:"error"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}
