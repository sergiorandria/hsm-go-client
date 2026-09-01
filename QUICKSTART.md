# HSM Go Client - Quick Start

Get up and running with the HSM Go client in 5 minutes.

## 1. Install the Library

```bash
go get github.com/sergiorandria/hsm-go-client
# For industrial PKCS#11, build with: CGO_ENABLED=1 go build
```

## 2. Create a Basic Program

### Microcontroller HTTP (ESP32/Raspberry Pi/self-made)

```go
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"github.com/sergiorandria/hsm-go-client/hsm/http"
)

func main() {
	// Connect to microcontroller HSM (ESP32, Pi, etc.)
	client := http.NewClient(http.Config{
		BaseURL:     "http://192.168.0.102",
		BearerToken: "your-token-here",
	})

	ctx := context.Background()

	// Create a key
	fmt.Println("Step 1: Creating key...")
	keyResp, err := client.CreateKey(ctx, "alice", false)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Key created")

	// Sign a file
	fmt.Println("\nStep 2: Signing file...")
	fileData := bytes.NewReader([]byte("Hello, HSM!"))
	signResp, err := client.SignFile(ctx, "alice", "message.txt", fileData, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ File signed!")
	fmt.Printf("\nResults:\n")
	fmt.Printf("  Hash: %s\n", signResp.HashHex)
	fmt.Printf("  Signature: %s\n", signResp.SignatureBase64)
}
```

### Industrial PKCS#11 (single USB/network device, `CGO_ENABLED=1`)

```go
package main

import (
	"context"
	"crypto/sha256"
	"fmt"
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
	fmt.Println(ki.PublicKeyPEM)
	digest := sha256.Sum256([]byte("Hello, HSM!"))
	sig, _ := driver.Sign(ctx, hsm.KeyID{Label: "alice"}, digest[:], hsm.MechanismECDSASHA256)
	fmt.Printf("Signature len %d\n", len(sig))
}
```

## 3. Run It

```bash
go run main.go
```

Output:
```
Step 1: Creating key...
✓ Key created

Step 2: Signing file...
✓ File signed!

Results:
  Hash: a1b2c3d4e5f6...
  Signature: MEUCIQDx+ZKL...
```

## Common Operations

### Create Keys for Multiple Users

```go
users := []string{"alice", "bob", "charlie"}
for _, user := range users {
	resp, err := client.CreateKey(ctx, user, false)
	if err != nil {
		log.Printf("Error creating key for %s: %v", user, err)
		continue
	}
	fmt.Printf("✓ Created key for %s\n", user)
}
```

### Sign a File from Disk

```go
file, err := os.Open("document.pdf")
if err != nil {
	log.Fatal(err)
}
defer file.Close()

resp, err := client.SignFile(ctx, "alice", "document.pdf", file, nil)
if err != nil {
	log.Fatal(err)
}

fmt.Printf("Signed: %s\n", resp.HashHex)
```

### Handle Authentication Errors

```go
resp, err := client.CreateKey(ctx, "user", false)
if err != nil {
	if strings.Contains(err.Error(), "unauthorized") {
		log.Fatal("Invalid auth token")
	}
	if strings.Contains(err.Error(), "key_already_exists") {
		log.Fatal("Key already exists")
	}
	log.Fatal(err)
}
```

### Add Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

resp, err := client.Sign(ctx, "alice", "file.txt", nil)
if err != nil {
	log.Fatal(err)
}
```

## Configuration

```go
client := hsm.NewClient(hsm.Config{
	// Device URL (required)
	BaseURL: "http://192.168.0.102",
	
	// Authentication token
	BearerToken: "my-secret-token",
	
	// Upload chunk size (optional, default 8KB)
	ChunkSize: 8 * 1024,
	
	// Request timeout (optional, default 30s)
	Timeout: 30 * time.Second,
})
```

## Running Examples

```bash
# Microcontroller HTTP
go run examples/basic_sign.go          # uses hsm/http (also hsm.NewClient shim)
go run examples/batch_signing.go
go run examples/key_management.go

# Industrial PKCS#11 (requires CGO_ENABLED=1 and SoftHSM2)
SOFTHSM2_CONF=/tmp/softhsm2.conf go run examples/pkcs11_sign.go
# Setup once: mkdir -p /tmp/softhsm_tokens; echo "directories.tokendir = /tmp/softhsm_tokens" > /tmp/softhsm2.conf; SOFTHSM2_CONF=/tmp/softhsm2.conf softhsm2-util --init-token --slot 0 --label test-token --pin 1234 --so-pin 1234
```

## Troubleshooting

### "connection refused"
- Check HSM is running on 192.168.0.102
- Verify network connectivity

### "unauthorized"
- Check BearerToken matches HSM configuration
- Verify auth is enabled on device

### "rate_limited"
- Slow down requests
- Implement retry with exponential backoff

### "file_too_large"
- Maximum file size is 32 KiB
- Split large files into smaller chunks

## Next Steps

- Read the [full documentation](README.md)
- Check [API reference](API_REFERENCE.md)
- See [integration examples](INTEGRATION.md)
- Review [usage guide](USAGE.md)

## Testing

Run the test suite:

```bash
go test ./... -v
```

Run with coverage:

```bash
go test ./... -cover
```

## Building CLI Tool

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sergiorandria/hsm-go-client/hsm"
)

func main() {
	userID := flag.String("user", "", "User ID")
	file := flag.String("file", "", "File to sign")
	token := flag.String("token", "", "API token")
	flag.Parse()

	if *userID == "" || *file == "" || *token == "" {
		flag.Usage()
		os.Exit(1)
	}

	client := hsm.NewClient(hsm.Config{
		BaseURL:     "http://192.168.0.102",
		BearerToken: *token,
	})

	f, err := os.Open(*file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	resp, err := client.SignFile(context.Background(), *userID, *file, f, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Hash: %s\n", resp.HashHex)
	fmt.Printf("Signature: %s\n", resp.SignatureBase64)
}
```

Build:
```bash
go build -o hsm-sign main.go

./hsm-sign -user alice -file document.pdf -token "your-token"
```

## Real-World Example: REST API

```go
package main

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/sergiorandria/hsm-go-client/hsm"
)

var hsmClient *hsm.Client

type SignRequest struct {
	UserID   string `json:"user_id"`
	Filename string `json:"filename"`
}

type SignResponse struct {
	Hash      string `json:"hash"`
	Signature string `json:"signature"`
}

func signHandler(w http.ResponseWriter, r *http.Request) {
	var req SignRequest
	json.NewDecoder(r.Body).Decode(&req)

	// For demo: read file from request body or disk
	resp, err := hsmClient.Sign(r.Context(), req.UserID, req.Filename, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SignResponse{
		Hash:      resp.HashHex,
		Signature: resp.SignatureBase64,
	})
}

func main() {
	hsmClient = hsm.NewClient(hsm.Config{
		BaseURL:     "http://192.168.0.102",
		BearerToken: os.Getenv("HSM_TOKEN"),
	})

	http.HandleFunc("/sign", signHandler)
	http.ListenAndServe(":8080", nil)
}
```

Start the server:
```bash
export HSM_TOKEN="your-token"
go run main.go

# Test it
curl -X POST http://localhost:8080/sign \
  -H "Content-Type: application/json" \
  -d '{"user_id":"alice","filename":"doc.pdf"}'
```

## Performance Tips

1. **Reuse client**: Create once, use many times
2. **Batch operations**: Group related operations
3. **Concurrent requests**: Safe to use with goroutines
4. **Timeouts**: Set appropriate timeouts for reliability

## Need Help?

- Check [USAGE.md](USAGE.md) for detailed examples
- See [API_REFERENCE.md](API_REFERENCE.md) for full API
- Review [INTEGRATION.md](INTEGRATION.md) for patterns
- Look at [examples/](examples/) directory

Happy signing! 🔐
