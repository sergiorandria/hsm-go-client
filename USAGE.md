# HSM Go Client - Usage Guide

Complete guide to using the HSM Go client library with your ESP32 HSM.

## Installation

```bash
go get github.com/sergiorandria/hsm-go-client
```

## Basic Setup

```go
package main

import (
    "context"
    "github.com/sergiorandria/hsm-go-client/hsm"
)

func main() {
    // Create client with default settings (connects to 192.168.0.102)
    client := hsm.NewClient(hsm.Config{
        BaseURL:     "http://192.168.0.102",
        BearerToken: "your-api-token", // Leave empty if auth is disabled
    })
    
    ctx := context.Background()
    
    // Now use the client...
}
```

## Common Operations

### 1. Create a Key

```go
resp, err := client.CreateKey(ctx, "alice", false)
if err != nil {
    log.Fatal(err)
}

fmt.Println("Public Key:")
fmt.Println(resp.PublicKeyPEM)
```

**Options:**
- `userID`: Unique identifier for the user (alphanumeric, up to 64 chars)
- `overwrite`: Set to `true` to replace an existing key

**Returns:**
- `PublicKeyAlgorithm`: "ECDSA-P256"
- `PublicKeyPEM`: Public key in PEM format (for external verification)

### 2. Upload a File

```go
file, err := os.Open("document.pdf")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

err = client.UploadFile(ctx, "document.pdf", file)
if err != nil {
    log.Fatal(err)
}
```

**File upload is chunked automatically** (8 KiB per chunk by default).

### 3. Sign a File

Sign a previously uploaded file:

```go
resp, err := client.Sign(ctx, "alice", "document.pdf", map[string]string{
    "purpose": "contract_execution",
    "date":    "2024-01-15",
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Hash: %s\n", resp.HashHex)
fmt.Printf("Signature: %s\n", resp.SignatureBase64)
fmt.Printf("Algorithm: %s\n", resp.SignatureAlgorithm) // ECDSA-SHA256
```

### 4. Combined Upload + Sign

For convenience, upload and sign in one call:

```go
file, _ := os.Open("contract.pdf")
defer file.Close()

resp, err := client.SignFile(ctx, "alice", "contract.pdf", file, map[string]interface{}{
    "type": "legal_document",
})
if err != nil {
    log.Fatal(err)
}
```

## Advanced Features

### Custom Configuration

```go
client := hsm.NewClient(hsm.Config{
    BaseURL:     "http://192.168.0.102",
    BearerToken: "my-secret-token",
    ChunkSize:   16 * 1024,      // 16 KiB chunks (max 32 KiB on ESP32)
    Timeout:     60 * time.Second,
})
```

### Context Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

resp, err := client.Sign(ctx, "alice", "file.txt", nil)
```

### Error Handling

```go
resp, err := client.CreateKey(ctx, "bob", false)
if err != nil {
    // Check specific error types
    if strings.Contains(err.Error(), "key_already_exists") {
        // Key exists - use overwrite:true or pick another user
    } else if strings.Contains(err.Error(), "unauthorized") {
        // Auth token is invalid
    } else if strings.Contains(err.Error(), "rate_limited") {
        // Too many requests - slow down
    }
}
```

### Concurrent Operations

The client is safe for concurrent use:

```go
var wg sync.WaitGroup

for _, userID := range []string{"alice", "bob", "charlie"} {
    wg.Add(1)
    go func(uid string) {
        defer wg.Done()
        resp, err := client.CreateKey(ctx, uid, false)
        // ...
    }(userID)
}

wg.Wait()
```

## Real-World Examples

### Example 1: Batch Document Signing

```go
func signDocuments(client *hsm.Client, userID string, files []string) error {
    for _, filepath := range files {
        file, err := os.Open(filepath)
        if err != nil {
            return err
        }
        defer file.Close()
        
        resp, err := client.SignFile(ctx, userID, filepath, file, nil)
        if err != nil {
            return err
        }
        
        fmt.Printf("Signed %s -> hash: %s\n", filepath, resp.HashHex)
    }
    return nil
}
```

### Example 2: Key Management Service

```go
type KeyManager struct {
    client *hsm.Client
    users  map[string]bool
}

func (km *KeyManager) EnsureUserKey(ctx context.Context, userID string) error {
    if km.users[userID] {
        return nil // Key already exists
    }
    
    resp, err := km.client.CreateKey(ctx, userID, false)
    if err != nil && strings.Contains(err.Error(), "key_already_exists") {
        km.users[userID] = true
        return nil
    }
    
    if err != nil {
        return err
    }
    
    km.users[userID] = true
    fmt.Printf("Created key for %s\n", userID)
    return nil
}
```

### Example 3: Document Verification

```go
func verifySignature(publicKeyPEM string, documentPath string, signatureBase64 string) bool {
    // 1. Parse public key
    block, _ := pem.Decode([]byte(publicKeyPEM))
    pub, _ := x509.ParsePKIXPublicKey(block.Bytes)
    
    // 2. Read document and compute hash
    data, _ := ioutil.ReadFile(documentPath)
    hash := sha256.Sum256(data)
    
    // 3. Decode signature from base64
    signature, _ := base64.StdEncoding.DecodeString(signatureBase64)
    
    // 4. Verify signature
    pubKey := pub.(*ecdsa.PublicKey)
    return ecdsa.VerifyASN1(pubKey, hash[:], signature)
}
```

### Example 4: REST API Endpoint

```go
func signHandler(w http.ResponseWriter, r *http.Request) {
    userID := r.FormValue("user_id")
    
    // Limit to 100MB
    r.ParseMultipartForm(100 * 1024 * 1024)
    
    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "Missing file", http.StatusBadRequest)
        return
    }
    defer file.Close()
    
    // Sign with HSM
    resp, err := client.SignFile(r.Context(), userID, header.Filename, file, nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Return signature
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}
```

## Verification

### Using OpenSSL

```bash
# 1. Save the public key
echo "$PUBLIC_KEY_PEM" > public.pem

# 2. Decode the signature
echo "$SIGNATURE_BASE64" | base64 -d > signature.der

# 3. Verify
openssl dgst -sha256 -verify public.pem -signature signature.der document.pdf
```

### Using Go

```go
import (
    "crypto/ecdsa"
    "crypto/sha256"
    "crypto/x509"
    "encoding/base64"
    "encoding/pem"
)

func verifySignature(pubKeyPEM string, docPath string, sigBase64 string) (bool, error) {
    // Parse public key
    block, _ := pem.Decode([]byte(pubKeyPEM))
    pub, _ := x509.ParsePKIXPublicKey(block.Bytes)
    pubKey := pub.(*ecdsa.PublicKey)
    
    // Hash document
    data, _ := ioutil.ReadFile(docPath)
    hash := sha256.Sum256(data)
    
    // Decode and verify signature
    sig, _ := base64.StdEncoding.DecodeString(sigBase64)
    return ecdsa.VerifyASN1(pubKey, hash[:], sig), nil
}
```

## Error Reference

| Error | Meaning | Action |
|-------|---------|--------|
| `key_already_exists` | User key exists | Use `overwrite: true` or choose different user |
| `unauthorized` | Invalid/missing token | Check `BearerToken` config |
| `rate_limited` | Too many requests | Implement exponential backoff |
| `file_too_large` | File > 32 KiB | Split into smaller files |
| `unknown_userId` | User has no key | Create key first with `CreateKey()` |
| `file_not_found` | File not uploaded | Run `UploadFile()` before `Sign()` |

## Performance Tips

1. **Reuse client**: Create once, use many times
2. **Batch operations**: Group multiple signs per user
3. **Chunk size**: Adjust `ChunkSize` based on network
4. **Timeouts**: Set appropriate timeouts for your use case
5. **Connection pooling**: HTTP client uses connection pooling automatically

## Security Considerations

1. **Token security**: Protect your `BearerToken` like a password
2. **TLS**: Use HTTPS in production: `https://your-hsm-domain.com`
3. **Key rotation**: Plan for periodic key updates
4. **Audit logging**: Log all signing operations
5. **Network isolation**: Keep HSM on protected network

## Troubleshooting

### Connection refused
```
Error: http request failed: connection refused
→ Check HSM is running and IP is correct
```

### Unauthorized
```
Error: unauthorized
→ Verify BearerToken matches HSM configuration
```

### Timeout
```
Error: context deadline exceeded
→ Increase timeout or check network performance
```

### File too large
```
Error: file_too_large
→ Files limited to 32 KiB, split and sign separately
```

## Testing

```bash
# Run tests
go test -v ./hsm

# With coverage
go test -cover ./hsm

# Benchmarks
go test -bench=. -benchmem ./hsm
```

## Building Examples

```bash
cd examples
go build basic_sign.go
go build batch_signing.go
go build key_management.go

./basic_sign
./batch_signing
./key_management
```
