# HSM Go Client

[![CI](https://github.com/sergiorandria/hsm-go-client/actions/workflows/ci.yml/badge.svg)](https://github.com/sergiorandria/hsm-go-client/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/sergiorandria/hsm-go-client.svg)](https://pkg.go.dev/github.com/sergiorandria/hsm-go-client)
[![Go Version](https://img.shields.io/github/go-mod/go-version/sergiorandria/hsm-go-client)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A production-ready Go client library for HSMs: microcontroller HTTP (ESP32/Raspberry Pi/self-made) and industrial PKCS#11 (Thales Luna, nShield, Utimaco, YubiHSM2, AWS CloudHSM, SoftHSM2).

## Features

- **Backends**: `hsm/http` for microcontrollers (ESP32/Raspberry Pi/HTTP JSON) and `hsm/pkcs11` generic industrial; vendor adapters `hsm/yubihsm`, `hsm/luna`, `hsm/cloudhsm` for contradictions
- **Generic Driver**: `hsm.Driver` interface + `hsm.NewDriver` registry + `crypto.Signer` (no change to generic when adapting)
- **Key Generation**: ECDSA P-256/P-384/P-521, RSA 2048-4096, Ed25519 via PKCS#11; ECDSA P-256 via HTTP
- **Signing**: File upload + sign (HTTP) or host-hash + sign digest (PKCS#11)
- **Production**: Sentinel errors `Err...`, `WithLogger`/`WithMetrics`/`WithRetry`, `Middleware` (logging/metrics/retry), `Validate()`, `Health()` probes, `SecureString` PIN, mTLS `TLSConfig`, `CircuitBreaker`/`RateLimiter`, `InMemoryMetrics`
- **Ops**: Dockerfile (CGO), `docker-compose` SoftHSM, `cmd/hsm-cli`, `deploy/k8s` with Secret + probes
- **Testing**: SoftHSM2 integration, `go test -race`, `SOFTHSM2_CONF`

## Installation

```bash
go get github.com/sergiorandria/hsm-go-client
```

## Quick Start

### Microcontroller HTTP (ESP32/Raspberry Pi/self-made)

```go
package main

import (
    "bytes"
    "context"
    "log"
    
    "github.com/sergiorandria/hsm-go-client/hsm"
    "github.com/sergiorandria/hsm-go-client/hsm/http" // canonical; hsm.NewClient shim also works
)

func main() {
    // Generic driver (or use http.NewClient directly)
    client := http.NewClient(http.Config{
        BaseURL:     "http://192.168.0.102",
        BearerToken: "your-token-here",
    })
    // Or legacy shim: hsm.NewClient(hsm.Config{...})
    
    ctx := context.Background()
    
    // Create a key for a user
    keyResp, err := client.CreateKey(ctx, "alice", false)
    if err != nil {
        log.Fatal(err)
    }
    
    // Sign a file
    fileData := bytes.NewReader([]byte("Hello, World!"))
    signResp, err := client.SignFile(ctx, "alice", "message.txt", fileData, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Signature: %s", signResp.SignatureBase64)
}
```

### Industrial PKCS#11 (Thales Luna, nShield, SoftHSM2) — requires `CGO_ENABLED=1`

```go
package main

import (
    "context"
    "crypto/sha256"
    "log"
    "github.com/sergiorandria/hsm-go-client/hsm"
)

func main() {
    driver, err := hsm.NewDriver(hsm.DriverConfig{
        Backend: "pkcs11",
        PKCS11: hsm.PKCS11Config{
            LibraryPath: "/usr/lib/softhsm/libsofthsm2.so",
            TokenLabel:  "test-token",
            PIN:         "1234",
        },
    })
    if err != nil { log.Fatal(err) }
    defer driver.Close()
    ctx := context.Background()
    ki, _ := driver.GenerateKey(ctx, hsm.KeySpec{Label: "alice", Curve: "P-256"})
    log.Println(ki.PublicKeyPEM)
    digest := sha256.Sum256([]byte("Hello, World!"))
    sig, _ := driver.Sign(ctx, hsm.KeyID{Label: "alice"}, digest[:], hsm.MechanismECDSASHA256)
    log.Printf("Signature len %d", len(sig))
}
```

## API Overview

### Creating Keys

```go
// Create a new ECDSA P-256 key for a user
resp, err := client.CreateKey(ctx, "user_id", false)
if err != nil {
    log.Fatal(err)
}
fmt.Println(resp.PublicKeyPEM) // Public key in PEM format
```

### Uploading Files

```go
// Upload a file in chunks
file, _ := os.Open("document.pdf")
defer file.Close()

err := client.UploadFile(ctx, "document.pdf", file)
if err != nil {
    log.Fatal(err)
}
```

### Signing

```go
// Sign a previously uploaded file
resp, err := client.Sign(ctx, "user_id", "document.pdf", metadata)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Hash: %s\n", resp.HashHex)
fmt.Printf("Signature: %s\n", resp.SignatureBase64)
```

### Combined Upload and Sign

```go
// Upload and sign in one call
file, _ := os.Open("document.pdf")
defer file.Close()

resp, err := client.SignFile(ctx, "user_id", "document.pdf", file, metadata)
if err != nil {
    log.Fatal(err)
}
```

## Configuration

```go
client := hsm.NewClient(hsm.Config{
    // Base URL of the HSM (default: "http://192.168.0.102")
    BaseURL: "http://192.168.0.102",
    
    // Bearer token for authentication
    BearerToken: "your-token-here",
    
    // Custom HTTP client (optional)
    HTTPClient: &http.Client{},
    
    // Chunk size for file uploads (default: 8 KiB)
    ChunkSize: 8 * 1024,
    
    // Request timeout (default: 30s)
    Timeout: 30 * time.Second,
})
```

## Context and Cancellation

All operations support context-based cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

resp, err := client.CreateKey(ctx, "user_id", false)
```

## Error Handling

The client returns detailed error messages from the HSM:

```go
resp, err := client.CreateKey(ctx, "user_id", false)
if err != nil {
    if strings.Contains(err.Error(), "key_already_exists") {
        // Handle key already exists
    }
}
```

## Examples

### Basic Signing
See `examples/basic_sign.go`

### Batch Signing
See `examples/batch_signing.go`

### Key Management
See `examples/key_management.go`

## Testing

Run tests:
```bash
go test ./...
```

Run with coverage:
```bash
go test -cover ./...
```

Run benchmarks:
```bash
go test -bench=. ./hsm
```

## Use Cases

This library can be used in:

- **Document Signing Applications**: Sign PDFs, contracts, or any files
- **Key Management Services**: Build custom key management solutions
- **Digital Signature Providers**: Create digital signature services
- **IoT Applications**: Sign data from edge devices
- **Compliance Systems**: Maintain audit trails of signed documents
- **API Gateways**: Add cryptographic signing to API calls

## Verification

To verify a signature outside of the HSM:

```bash
# 1. Get the public key from HSM response
# 2. Save the file
# 3. Decode the signature from base64
base64 -d signature.b64 > signature.der

# 4. Verify using OpenSSL
openssl dgst -sha256 -verify public.pem -signature signature.der file.dat
```

## Backends

| Backend | Package | Transport | Use for |
|---------|---------|-----------|---------|
| Microcontroller HTTP | `hsm/http` (shim `hsm` + deprecated `hsm/esp32`) | HTTP `POST /cmd` JSON, chunked 8 KiB | ESP32, Raspberry Pi, self-made boards |
| PKCS#11 generic | `hsm/pkcs11` via `hsm.Driver` (`CGO_ENABLED=1`) | PKCS#11 `libsofthsm2.so`/`libCryptoki2.so` | SoftHSM2, any PKCS#11 |
| YubiHSM2 adapter | `hsm/yubihsm` (`CGO_ENABLED=1`) | PKCS#11 `yubihsm_pkcs11.so` + `yubihsm-connector` | YubiHSM2 USB (PIN `0001:password`) |
| Luna adapter | `hsm/luna` (`CGO_ENABLED=1`) | PKCS#11 `libCryptoki2_64.so` | Thales Luna Network HSM (partition) |
| CloudHSM adapter | `hsm/cloudhsm` (`CGO_ENABLED=1`) | PKCS#11 `libcloudhsm_pkcs11.so` | AWS CloudHSM v5 (CU `user:pass`, cluster) |

Generic `hsm.Driver` (`GenerateKey`, `GetPublicKey`, `Sign(digest)`, `Signer() crypto.Signer`, `ListKeys`, `Info`, `Close`) works for all; vendor adapters wrap `hsm/pkcs11` without modifying it (interface+adapter for contradictions: YubiHSM PIN format, Luna PSS params/partition, CloudHSM CU/cluster sync). HTTP signs via file upload, PKCS#11 signs host-computed digest (single USB/network device, session pool).

## Production Ready

```go
// Secure PIN from env, mTLS, validation, middleware, health
pin := hsm.PINFromEnv() // HSM_PIN
tokenLabel := os.Getenv("PKCS11_TOKEN")
driver, err := hsm.NewDriver(hsm.DriverConfig{
    Backend: "cloudhsm",
    PKCS11: hsm.PKCS11Config{LibraryPath: "/opt/cloudhsm/lib/libcloudhsm_pkcs11.so", TokenLabel: tokenLabel, PIN: string(pin)},
}, hsm.WithLogger(slog.Default()), hsm.WithRetry(3, 200*time.Millisecond), hsm.WithMetrics(func(op string, d time.Duration, err error){ metrics.Observe(op,d,err)}))
if err != nil { log.Fatal(err) }
defer driver.Close()
// Health for k8s probe
status, _ := hsm.Health(ctx, driver, "cloudhsm") // slot, token, latency
// Resilient wrapper for HA
rcb := hsm.NewCircuitBreaker(5, 30*time.Second)
rl := hsm.NewRateLimiter(10, 10)
resilient := hsm.NewResilientDriver(driver, rcb, rl, 5*time.Second)
// Ops: docker-compose up, cmd/hsm-cli -backend pkcs11 -action health
```

See `Dockerfile`, `docker-compose.yml` (SoftHSM), `softhsm2.conf`, `cmd/hsm-cli`, `deploy/k8s/deployment.yaml` (Secret for PIN, liveness/readiness).

## Implementation Details

- **HTTP Chunk Size**: 8 KiB (mTLS via `http.Config.TLSConfig` / `DriverConfig.HTTP.TLSConfig`)
- **HTTP Authentication**: Bearer token (via `hsm.BearerTokenFromEnv`, `SecureString`, `Redact`) + mTLS
- **PKCS#11 Authentication**: `PIN` via `hsm.PINFromEnv` (`SecureString`), `TokenLabel`/`SlotID`/`LibraryPath`
- **Protocols**: HTTP/HTTPS (microcontroller) / PKCS#11 (industrial)
- **Encoding**: Base64, JSON (HTTP)
- **Thread-safe**: Session pool + `CircuitBreaker`/`RateLimiter` + retry middleware

## Debugging

Enable verbose logging by checking HTTP responses:

```go
// Create a custom HTTP client with logging
transport := &http.Transport{
    // ... configuration ...
}

// Use a custom HTTP client
httpClient := &http.Client{Transport: transport}

client := hsm.NewClient(hsm.Config{
    HTTPClient: httpClient,
})
```

## License

MIT — see [LICENSE](LICENSE). Part of the Virtual HSM (vHSM) system.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, `make` targets, and PR guidelines.

## Support

For issues or feature requests, please [open an issue](https://github.com/sergiorandria/hsm-go-client/issues/new/choose).
