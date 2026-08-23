# HSM Go Client

A production-ready Go client library for interacting with the ESP32 HTTP-based HSM (Hardware Security Module).

## Features

- **Key Generation**: Create ECDSA P-256 keys for users
- **File Upload**: Efficiently upload files in chunks to the HSM
- **Signing**: Sign files with user keys
- **Error Handling**: Comprehensive error handling with detailed error messages
- **Context Support**: Full support for request cancellation and timeouts
- **Concurrent Operations**: Safe for concurrent use
- **Testing**: Comprehensive test suite with benchmarks

## Installation

```bash
go get github.com/sergio/hsm-go-client
```

## Quick Start

```go
package main

import (
    "bytes"
    "context"
    "log"
    
    "github.com/sergio/hsm-go-client/hsm"
)

func main() {
    client := hsm.NewClient(hsm.Config{
        BaseURL:     "http://192.168.0.102",
        BearerToken: "your-token-here",
    })
    
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

## Implementation Details

- **Chunk Size**: Maximum upload chunk is 8 KiB (configurable, but ESP32 limits to 32 KiB total)
- **Authentication**: Bearer token in Authorization header
- **Protocols**: HTTP/HTTPS
- **Encoding**: Base64 for binary data, JSON for all messages
- **Thread-safe**: Safe for concurrent goroutines

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

This project is part of the Virtual HSM (vHSM) system.

## Support

For issues or feature requests, refer to the main vHSM repository.
