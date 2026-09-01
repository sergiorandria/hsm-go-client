# HSM Go Client - API Reference

Complete API reference for the HSM Go client library.

## Package: `github.com/sergiorandria/hsm-go-client/hsm`

### Types

#### Client

```go
type Client struct {
    // Unexported fields
}

func NewClient(cfg Config) *Client
```

The main client for communicating with the ESP32 HSM.

#### Config

```go
type Config struct {
    BaseURL     string        // Base URL of HSM (default: "http://192.168.0.102")
    BearerToken string        // Authentication token
    HTTPClient  *http.Client  // Custom HTTP client (optional)
    ChunkSize   int           // Upload chunk size (default: 8192)
    Timeout     time.Duration // Request timeout (default: 30s)
}
```

Configuration for creating a new Client.

#### CreateKeyResponse

```go
type CreateKeyResponse struct {
    Status             string // "ok" or "error"
    UserID             string // User identifier
    PublicKeyAlgorithm string // "ECDSA-P256"
    PublicKeyPEM       string // Public key in PEM format
    Error              string // Error code (if status != "ok")
    Message            string // Error message
    Hint               string // Error hint
}
```

Response from CreateKey operation.

#### UploadStartResponse

```go
type UploadStartResponse struct {
    Status   string // "ok" or "error"
    Filename string // Uploaded filename
    Error    string // Error code
    Message  string // Error message
    Hint     string // Error hint
}
```

Response from UploadStart operation.

#### UploadChunkResponse

```go
type UploadChunkResponse struct {
    Status       string // "ok" or "error"
    Filename     string // Filename
    BytesWritten int64  // Bytes written in this chunk
    TotalBytes   int64  // Total bytes uploaded so far
    Error        string // Error code
    Message      string // Error message
    Hint         string // Error hint
}
```

Response from UploadChunk operation.

#### UploadEndResponse

```go
type UploadEndResponse struct {
    Status    string // "ok" or "error"
    Filename  string // Filename
    TotalBytes int64  // Total file size
    Error     string // Error code
    Message   string // Error message
    Hint      string // Error hint
}
```

Response from UploadEnd operation.

#### SignResponse

```go
type SignResponse struct {
    Status             string // "ok" or "error"
    UserID             string // User identifier
    Filename           string // Filename that was signed
    HashAlgorithm      string // "SHA-256"
    HashHex            string // Hash in hex format
    SignatureAlgorithm string // "ECDSA-SHA256"
    SignatureBase64    string // Signature in base64 format
    Error              string // Error code
    Message            string // Error message
    Hint               string // Error hint
}
```

Response from Sign operation.

#### Signature

```go
type Signature struct {
    Algorithm string
    Base64    string
}

func (s *Signature) Bytes() ([]byte, error)
```

Represents a cryptographic signature with Base64 decoding capability.

#### Hash

```go
type Hash struct {
    Algorithm string // "SHA-256"
    Hex       string // Hexadecimal representation
}
```

Represents a hash value.

#### Key

```go
type Key struct {
    UserID       string
    Algorithm    string
    PublicKeyPEM string
    CreatedAt    string
    ExpiresAt    string
}
```

Represents a cryptographic key.

#### SignatureBundle

```go
type SignatureBundle struct {
    UserID    string
    Filename  string
    Hash      Hash
    Signature Signature
    SignedAt  string
    Metadata  interface{}
}
```

Complete result of a signing operation.

#### ErrorResponse

```go
type ErrorResponse struct {
    Status  string // "error"
    Error   string // Error code
    Message string // Error message
    Hint    string // Error hint
}
```

Represents an error response from the HSM.

### Methods

#### NewClient

```go
func NewClient(cfg Config) *Client
```

Creates a new HSM client. If Config fields are empty, defaults are used:
- BaseURL: "http://192.168.0.102"
- ChunkSize: 8 KiB
- Timeout: 30 seconds

**Example:**
```go
client := hsm.NewClient(hsm.Config{
    BaseURL:     "http://192.168.0.102",
    BearerToken: "my-token",
})
```

#### CreateKey

```go
func (c *Client) CreateKey(ctx context.Context, userID string, overwrite bool) (*CreateKeyResponse, error)
```

Generates a new ECDSA P-256 key for a user.

**Parameters:**
- `ctx`: Context for cancellation and timeouts
- `userID`: Unique user identifier (1-64 alphanumeric characters)
- `overwrite`: If true, replaces existing key; if false, returns error if key exists

**Returns:**
- `*CreateKeyResponse`: Key information including public key in PEM format
- `error`: Error if operation failed

**Errors:**
- `missing_or_invalid_userId`: Invalid userId format
- `key_already_exists`: Key already exists and overwrite=false
- `unauthorized`: Invalid authentication token

**Example:**
```go
resp, err := client.CreateKey(ctx, "alice", false)
if err != nil {
    log.Fatal(err)
}
fmt.Println(resp.PublicKeyPEM)
```

#### UploadStart

```go
func (c *Client) UploadStart(ctx context.Context, filename string) (*UploadStartResponse, error)
```

Initializes an upload session.

**Parameters:**
- `ctx`: Context
- `filename`: Name of file to upload (1-128 alphanumeric characters)

**Returns:**
- `*UploadStartResponse`: Confirmation of session start
- `error`: Error if failed

**Example:**
```go
resp, err := client.UploadStart(ctx, "document.pdf")
```

#### UploadChunk

```go
func (c *Client) UploadChunk(ctx context.Context, filename string, data []byte) (*UploadChunkResponse, error)
```

Uploads a chunk of file data.

**Parameters:**
- `ctx`: Context
- `filename`: Filename being uploaded
- `data`: Chunk data (automatically base64-encoded)

**Returns:**
- `*UploadChunkResponse`: Upload progress
- `error`: Error if failed

**Example:**
```go
chunk := make([]byte, 8192)
n, _ := file.Read(chunk)
resp, err := client.UploadChunk(ctx, "document.pdf", chunk[:n])
```

#### UploadEnd

```go
func (c *Client) UploadEnd(ctx context.Context, filename string) (*UploadEndResponse, error)
```

Finalizes an upload session.

**Parameters:**
- `ctx`: Context
- `filename`: Filename being uploaded

**Returns:**
- `*UploadEndResponse`: Final file information
- `error`: Error if failed

**Example:**
```go
resp, err := client.UploadEnd(ctx, "document.pdf")
```

#### UploadFile

```go
func (c *Client) UploadFile(ctx context.Context, filename string, data io.Reader) error
```

Convenience method that uploads a complete file in chunks.

**Parameters:**
- `ctx`: Context
- `filename`: Name of file
- `data`: io.Reader (File, bytes.Buffer, etc.)

**Returns:**
- `error`: Error if failed (or nil on success)

**Example:**
```go
file, _ := os.Open("document.pdf")
defer file.Close()
err := client.UploadFile(ctx, "document.pdf", file)
```

#### Sign

```go
func (c *Client) Sign(ctx context.Context, userID string, filename string, metadata interface{}) (*SignResponse, error)
```

Signs a previously uploaded file with a user's key.

**Parameters:**
- `ctx`: Context
- `userID`: User identifier
- `filename`: Name of uploaded file
- `metadata`: Optional metadata (not signed, echoed back)

**Returns:**
- `*SignResponse`: Signature information
- `error`: Error if failed

**Example:**
```go
resp, err := client.Sign(ctx, "alice", "document.pdf", map[string]string{
    "purpose": "contract",
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Signature: %s\n", resp.SignatureBase64)
```

#### SignFile

```go
func (c *Client) SignFile(ctx context.Context, userID string, filename string, data io.Reader, metadata interface{}) (*SignResponse, error)
```

Convenience method that uploads and signs a file in one call.

**Parameters:**
- `ctx`: Context
- `userID`: User identifier
- `filename`: Filename
- `data`: io.Reader
- `metadata`: Optional metadata

**Returns:**
- `*SignResponse`: Signature information
- `error`: Error if failed

**Example:**
```go
file, _ := os.Open("contract.pdf")
defer file.Close()
resp, err := client.SignFile(ctx, "alice", "contract.pdf", file, nil)
```

### Constants

```go
const (
    DefaultChunkSize = 8 * 1024 // 8 KiB
)
```

## Error Codes

| Error Code | HTTP Status | Description | Solution |
|-----------|------------|-----------|----------|
| invalid_json | 400 | Malformed JSON | Check request format |
| missing_cmd | 400 | Missing command | Include "cmd" field |
| missing_or_invalid_userId | 400 | Invalid userID | Use alphanumeric, 1-64 chars |
| missing_or_invalid_filename | 400 | Invalid filename | Use alphanumeric, 1-128 chars |
| missing_dataBase64 | 400 | Missing data | Include "dataBase64" field |
| invalid_base64 | 400 | Invalid base64 encoding | Re-encode data |
| body_too_large_or_empty | 400 | Request too large | Keep under 8 KiB |
| unauthorized | 401 | Invalid token | Check BearerToken |
| unknown_cmd | 404 | Unknown command | Use valid cmd |
| unknown_userId | 404 | User not found | Create key first |
| file_not_found | 404 | File not found | Upload file first |
| key_already_exists | 409 | Key exists | Set overwrite=true |
| file_too_large | 413 | File > 32 KiB | Split into smaller files |
| rate_limited | 429 | Too many requests | Slow down requests |
| temporarily_locked | 503 | Auth lockout | Wait for cooldown |

## Usage Patterns

### Pattern 1: Simple Signing

```go
func simpleSign(client *hsm.Client, userID, filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    resp, err := client.SignFile(context.Background(), userID, filePath, file, nil)
    if err != nil {
        return err
    }
    
    fmt.Printf("Hash: %s\nSignature: %s\n", resp.HashHex, resp.SignatureBase64)
    return nil
}
```

### Pattern 2: With Retry Logic

```go
func signWithRetry(client *hsm.Client, userID, filename string, maxRetries int) (*hsm.SignResponse, error) {
    var err error
    var resp *hsm.SignResponse
    
    for i := 0; i < maxRetries; i++ {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        resp, err = client.Sign(ctx, userID, filename, nil)
        cancel()
        
        if err == nil {
            return resp, nil
        }
        
        // Don't retry on auth errors
        if strings.Contains(err.Error(), "unauthorized") {
            return nil, err
        }
        
        // Exponential backoff
        time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second)
    }
    
    return nil, err
}
```

### Pattern 3: Batch Operations

```go
func signBatch(client *hsm.Client, userID string, files []string) (map[string]string, error) {
    results := make(map[string]string)
    
    for _, file := range files {
        f, err := os.Open(file)
        if err != nil {
            continue
        }
        
        resp, err := client.SignFile(context.Background(), userID, file, f, nil)
        f.Close()
        
        if err == nil {
            results[file] = resp.SignatureBase64
        }
    }
    
    return results, nil
}
```

## Thread Safety

The Client is safe for concurrent use:

```go
var wg sync.WaitGroup
for _, userID := range userIDs {
    wg.Add(1)
    go func(uid string) {
        defer wg.Done()
        client.CreateKey(context.Background(), uid, false)
    }(userID)
}
wg.Wait()
```

## Performance Characteristics

- **CreateKey**: ~500ms (key generation)
- **UploadFile**: ~10-50ms (8 KiB chunk)
- **Sign**: ~200-400ms (ECDSA signature)
- **SignFile**: ~300-500ms (combined)

Actual times depend on network and ESP32 load.
