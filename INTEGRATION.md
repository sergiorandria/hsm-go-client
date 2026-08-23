# Integration Guide - HSM Go Client

Guide to integrating the HSM Go client library into your applications.

## Application Types

### 1. Document Management System

```go
package main

import (
    "context"
    "github.com/sergio/hsm-go-client/hsm"
)

type DocumentService struct {
    hsmClient *hsm.Client
}

func (ds *DocumentService) SignDocument(ctx context.Context, docID, userID string) (signature, hash string, err error) {
    // Load document from database
    docPath := "/docs/" + docID + ".pdf"
    
    file, _ := os.Open(docPath)
    defer file.Close()
    
    resp, err := ds.hsmClient.SignFile(ctx, userID, docID, file, map[string]interface{}{
        "doc_id": docID,
        "type":   "pdf",
    })
    
    if err != nil {
        return "", "", err
    }
    
    // Store signature in database
    return resp.SignatureBase64, resp.HashHex, nil
}
```

**Integrations:**
- MongoDB: Store signatures + hashes
- PostgreSQL: Maintain audit trail
- S3: Store signed documents
- Elasticsearch: Index signatures

### 2. REST API Service

```go
package main

import (
    "encoding/json"
    "net/http"
    "github.com/sergio/hsm-go-client/hsm"
)

type SignRequest struct {
    UserID   string `json:"user_id"`
    Filename string `json:"filename"`
    Content  []byte `json:"content"`
}

type SignResponse struct {
    Hash      string `json:"hash"`
    Signature string `json:"signature"`
}

var hsmClient *hsm.Client

func signHandler(w http.ResponseWriter, r *http.Request) {
    var req SignRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    resp, err := hsmClient.SignFile(
        r.Context(),
        req.UserID,
        req.Filename,
        bytes.NewReader(req.Content),
        nil,
    )
    
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
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
    
    http.HandleFunc("/api/sign", signHandler)
    http.ListenAndServe(":8080", nil)
}
```

### 3. CLI Tool

```go
package main

import (
    "context"
    "flag"
    "github.com/sergio/hsm-go-client/hsm"
)

func main() {
    userID := flag.String("user", "", "User ID")
    filePath := flag.String("file", "", "File to sign")
    token := flag.String("token", "", "API token")
    flag.Parse()
    
    client := hsm.NewClient(hsm.Config{
        BaseURL:     "http://192.168.0.102",
        BearerToken: *token,
    })
    
    file, _ := os.Open(*filePath)
    defer file.Close()
    
    resp, err := client.SignFile(context.Background(), *userID, *filePath, file, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Signature: %s\n", resp.SignatureBase64)
    fmt.Printf("Hash: %s\n", resp.HashHex)
}
```

### 4. Microservice Architecture

```go
// Service using dependency injection
type SigningService struct {
    hsmClient *hsm.Client
    logger    *log.Logger
    cache     *Cache
}

func NewSigningService(cfg Config) *SigningService {
    client := hsm.NewClient(hsm.Config{
        BaseURL:     cfg.HSMBaseURL,
        BearerToken: cfg.HSMToken,
        Timeout:     cfg.RequestTimeout,
    })
    
    return &SigningService{
        hsmClient: client,
        logger:    cfg.Logger,
        cache:     NewCache(),
    }
}

func (s *SigningService) SignWithCaching(ctx context.Context, userID, filename string, data io.Reader) (*SignResult, error) {
    // Check cache
    if cached := s.cache.Get(filename); cached != nil {
        s.logger.Printf("Cache hit for %s", filename)
        return cached, nil
    }
    
    resp, err := s.hsmClient.SignFile(ctx, userID, filename, data, nil)
    if err != nil {
        s.logger.Printf("Signing failed: %v", err)
        return nil, err
    }
    
    // Cache result
    s.cache.Set(filename, resp)
    return resp, nil
}
```

### 5. Event-Driven System

```go
// Using message queues (e.g., RabbitMQ, Kafka)
type SigningConsumer struct {
    hsmClient *hsm.Client
    queue     MessageQueue
}

func (c *SigningConsumer) ProcessSigningRequests(ctx context.Context) {
    for {
        msg, err := c.queue.Receive(ctx)
        if err != nil {
            continue
        }
        
        var req SigningRequest
        json.Unmarshal(msg.Body, &req)
        
        resp, err := c.hsmClient.SignFile(ctx, req.UserID, req.Filename, req.Data, nil)
        
        // Publish result
        c.queue.Publish(SigningResult{
            RequestID: req.ID,
            Hash:      resp.HashHex,
            Signature: resp.SignatureBase64,
            Error:     err,
        })
    }
}
```

### 6. Batch Processing

```go
package main

import (
    "github.com/sergio/hsm-go-client/hsm"
    "sync"
)

type BatchSigner struct {
    client    *hsm.Client
    workers   int
    workQueue chan WorkItem
}

type WorkItem struct {
    UserID   string
    Filename string
    Path     string
    Result   chan<- Result
}

func (bs *BatchSigner) Start(ctx context.Context) {
    for i := 0; i < bs.workers; i++ {
        go bs.worker(ctx)
    }
}

func (bs *BatchSigner) worker(ctx context.Context) {
    for item := range bs.workQueue {
        file, _ := os.Open(item.Path)
        resp, err := bs.client.SignFile(ctx, item.UserID, item.Filename, file, nil)
        file.Close()
        
        item.Result <- Result{
            Hash:      resp.HashHex,
            Signature: resp.SignatureBase64,
            Error:     err,
        }
    }
}

func (bs *BatchSigner) SignMany(ctx context.Context, items []WorkItem) []Result {
    for _, item := range items {
        bs.workQueue <- item
    }
    
    results := make([]Result, len(items))
    for i := range items {
        results[i] = <-items[i].Result
    }
    return results
}
```

## Deployment Patterns

### Docker

```dockerfile
FROM golang:1.21-alpine

WORKDIR /app

COPY . .

RUN go build -o myapp .

ENV HSM_URL=http://hsm:8080
ENV HSM_TOKEN=your-token

CMD ["./myapp"]
```

### Kubernetes

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: hsm-config
data:
  HSM_URL: "http://hsm-service:8080"

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: signing-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: signing-service
  template:
    metadata:
      labels:
        app: signing-service
    spec:
      containers:
      - name: app
        image: myregistry/signing-service:latest
        env:
        - name: HSM_URL
          valueFrom:
            configMapKeyRef:
              name: hsm-config
              key: HSM_URL
        - name: HSM_TOKEN
          valueFrom:
            secretKeyRef:
              name: hsm-secret
              key: token
        ports:
        - containerPort: 8080
```

## Database Integration

### PostgreSQL

```go
type SignatureRecord struct {
    ID        int       `db:"id"`
    UserID    string    `db:"user_id"`
    DocumentID string   `db:"document_id"`
    Hash      string    `db:"hash"`
    Signature string    `db:"signature"`
    CreatedAt time.Time `db:"created_at"`
}

func (s *SigningService) SignAndStore(ctx context.Context, db *sql.DB, userID, docID string) error {
    resp, err := s.hsmClient.Sign(ctx, userID, docID, nil)
    if err != nil {
        return err
    }
    
    _, err = db.ExecContext(ctx,
        `INSERT INTO signatures (user_id, document_id, hash, signature, created_at)
         VALUES ($1, $2, $3, $4, $5)`,
        userID, docID, resp.HashHex, resp.SignatureBase64, time.Now())
    
    return err
}
```

### MongoDB

```go
func (s *SigningService) SignAndStoreAsync(ctx context.Context, coll *mongo.Collection, userID, docID string) error {
    resp, err := s.hsmClient.Sign(ctx, userID, docID, nil)
    if err != nil {
        return err
    }
    
    _, err = coll.InsertOne(ctx, bson.M{
        "user_id":     userID,
        "document_id": docID,
        "hash":        resp.HashHex,
        "signature":   resp.SignatureBase64,
        "timestamp":   time.Now(),
    })
    
    return err
}
```

## Testing Strategy

```go
func TestSigningService(t *testing.T) {
    // Mock HSM client
    mockClient := &MockHSMClient{
        SignFileFunc: func(ctx context.Context, userID, filename string, data io.Reader, metadata interface{}) (*hsm.SignResponse, error) {
            return &hsm.SignResponse{
                HashHex:       "abc123",
                SignatureBase64: "def456",
            }, nil
        },
    }
    
    service := &SigningService{hsmClient: mockClient}
    
    result, err := service.SignWithCaching(context.Background(), "user1", "doc.pdf", nil)
    
    assert.NoError(t, err)
    assert.Equal(t, "abc123", result.HashHex)
}
```

## Monitoring & Observability

```go
import "github.com/prometheus/client_golang/prometheus"

type MonitoredClient struct {
    client      *hsm.Client
    signCounter prometheus.Counter
    signLatency prometheus.Histogram
}

func (mc *MonitoredClient) Sign(ctx context.Context, userID, filename string, metadata interface{}) (*hsm.SignResponse, error) {
    start := time.Now()
    
    resp, err := mc.client.Sign(ctx, userID, filename, metadata)
    
    mc.signCounter.Inc()
    mc.signLatency.Observe(time.Since(start).Seconds())
    
    return resp, err
}
```

## Security Best Practices

1. **Environment Variables**
   ```go
   token := os.Getenv("HSM_API_TOKEN")
   hsmURL := os.Getenv("HSM_URL")
   ```

2. **Request Timeout**
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer cancel()
   ```

3. **Error Handling**
   ```go
   if err != nil {
       log.Printf("HSM error: %v", err)
       // Don't expose full error to client
       return "Internal error", http.StatusInternalServerError
   }
   ```

4. **TLS/HTTPS**
   ```go
   client := hsm.NewClient(hsm.Config{
       BaseURL: "https://hsm.example.com",
   })
   ```

## Performance Optimization

- Adjust `ChunkSize` for your network
- Implement connection pooling (automatic)
- Use request context timeouts
- Cache public keys locally
- Implement retry logic with backoff

## Troubleshooting Integration

See `USAGE.md` for detailed error reference and troubleshooting.
