package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultChunkSize is the default chunk size for file uploads (must be <= 32 KiB on microcontroller)
	DefaultChunkSize = 8 * 1024 // 8 KiB per chunk
)

// Client is the HTTP client for interacting with a microcontroller HSM
// (ESP32, Raspberry Pi, or any self-made board exposing POST /cmd JSON API).
type Client struct {
	baseURL     string
	bearerToken string
	httpClient  *http.Client
	chunkSize   int
}

// Config holds configuration for creating a new Client.
type Config struct {
	// BaseURL is the base URL of the HSM (e.g., "http://192.168.0.102")
	BaseURL string
	// BearerToken is the authentication token (if auth is enabled on the device)
	BearerToken string
	// HTTPClient is the underlying HTTP client to use (if nil, a default is created)
	HTTPClient *http.Client
	// ChunkSize is the size of chunks for file uploads (default: 8 KiB)
	ChunkSize int
	// Timeout is the timeout for individual requests (default: 30s)
	Timeout time.Duration
}

// NewClient creates a new microcontroller HTTP HSM client.
func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://192.168.0.102"
	}
	if cfg.ChunkSize == 0 {
		cfg.ChunkSize = DefaultChunkSize
	}
	if cfg.HTTPClient == nil {
		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		cfg.HTTPClient = &http.Client{Timeout: timeout}
	}

	// Ensure BaseURL doesn't have trailing slash
	cfg.BaseURL = strings.TrimSuffix(cfg.BaseURL, "/")

	return &Client{
		baseURL:     cfg.BaseURL,
		bearerToken: cfg.BearerToken,
		httpClient:  cfg.HTTPClient,
		chunkSize:   cfg.ChunkSize,
	}
}

// CreateKeyRequest is the request to create a new key.
type CreateKeyRequest struct {
	Cmd       string `json:"cmd"`
	UserID    string `json:"userId"`
	Overwrite bool   `json:"overwrite"`
}

// CreateKeyResponse is the response from a key creation request.
type CreateKeyResponse struct {
	Status             string `json:"status"`
	UserID             string `json:"userId"`
	PublicKeyAlgorithm string `json:"publicKeyAlgorithm"`
	PublicKeyPEM       string `json:"publicKeyPem"`
	Error              string `json:"error,omitempty"`
	Message            string `json:"message,omitempty"`
	Hint               string `json:"hint,omitempty"`
}

// CreateKey generates a new ECDSA P-256 key for a user.
func (c *Client) CreateKey(ctx context.Context, userID string, overwrite bool) (*CreateKeyResponse, error) {
	req := CreateKeyRequest{
		Cmd:       "createKey",
		UserID:    userID,
		Overwrite: overwrite,
	}
	resp := &CreateKeyResponse{}
	err := c.sendRequest(ctx, req, resp)
	if err != nil {
		return nil, err
	}
	if resp.Status != "ok" {
		return nil, fmt.Errorf("create key failed: %s - %s", resp.Error, resp.Message)
	}
	return resp, nil
}

// UploadStartRequest is the request to start an upload session.
type UploadStartRequest struct {
	Cmd      string `json:"cmd"`
	Filename string `json:"filename"`
}

// UploadStartResponse is the response from an upload start request.
type UploadStartResponse struct {
	Status   string `json:"status"`
	Filename string `json:"filename"`
	Error    string `json:"error,omitempty"`
	Message  string `json:"message,omitempty"`
	Hint     string `json:"hint,omitempty"`
}

// UploadStart initializes an upload session.
func (c *Client) UploadStart(ctx context.Context, filename string) (*UploadStartResponse, error) {
	req := UploadStartRequest{
		Cmd:      "uploadStart",
		Filename: filename,
	}
	resp := &UploadStartResponse{}
	err := c.sendRequest(ctx, req, resp)
	if err != nil {
		return nil, err
	}
	if resp.Status != "ok" {
		return nil, fmt.Errorf("upload start failed: %s - %s", resp.Error, resp.Message)
	}
	return resp, nil
}

// UploadChunkRequest is the request to upload a chunk of data.
type UploadChunkRequest struct {
	Cmd        string `json:"cmd"`
	Filename   string `json:"filename"`
	DataBase64 string `json:"dataBase64"`
}

// UploadChunkResponse is the response from an upload chunk request.
type UploadChunkResponse struct {
	Status       string `json:"status"`
	Filename     string `json:"filename"`
	BytesWritten int64  `json:"bytesWritten"`
	TotalBytes   int64  `json:"totalBytes"`
	Error        string `json:"error,omitempty"`
	Message      string `json:"message,omitempty"`
	Hint         string `json:"hint,omitempty"`
}

// UploadChunk uploads a chunk of file data.
func (c *Client) UploadChunk(ctx context.Context, filename string, data []byte) (*UploadChunkResponse, error) {
	b64Data := base64.StdEncoding.EncodeToString(data)
	req := UploadChunkRequest{
		Cmd:        "uploadChunk",
		Filename:   filename,
		DataBase64: b64Data,
	}
	resp := &UploadChunkResponse{}
	err := c.sendRequest(ctx, req, resp)
	if err != nil {
		return nil, err
	}
	if resp.Status != "ok" {
		return nil, fmt.Errorf("upload chunk failed: %s - %s", resp.Error, resp.Message)
	}
	return resp, nil
}

// UploadEndRequest is the request to finalize an upload session.
type UploadEndRequest struct {
	Cmd      string `json:"cmd"`
	Filename string `json:"filename"`
}

// UploadEndResponse is the response from an upload end request.
type UploadEndResponse struct {
	Status     string `json:"status"`
	Filename   string `json:"filename"`
	TotalBytes int64  `json:"totalBytes"`
	Error      string `json:"error,omitempty"`
	Message    string `json:"message,omitempty"`
	Hint       string `json:"hint,omitempty"`
}

// UploadEnd finalizes an upload session.
func (c *Client) UploadEnd(ctx context.Context, filename string) (*UploadEndResponse, error) {
	req := UploadEndRequest{
		Cmd:      "uploadEnd",
		Filename: filename,
	}
	resp := &UploadEndResponse{}
	err := c.sendRequest(ctx, req, resp)
	if err != nil {
		return nil, err
	}
	if resp.Status != "ok" {
		return nil, fmt.Errorf("upload end failed: %s - %s", resp.Error, resp.Message)
	}
	return resp, nil
}

// UploadFile uploads a complete file by chunking it.
func (c *Client) UploadFile(ctx context.Context, filename string, data io.Reader) error {
	if _, err := c.UploadStart(ctx, filename); err != nil {
		return fmt.Errorf("upload start: %w", err)
	}

	buf := make([]byte, c.chunkSize)
	for {
		n, err := data.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("read error: %w", err)
		}
		if n == 0 {
			break
		}

		if _, err := c.UploadChunk(ctx, filename, buf[:n]); err != nil {
			return fmt.Errorf("upload chunk: %w", err)
		}
	}

	if _, err := c.UploadEnd(ctx, filename); err != nil {
		return fmt.Errorf("upload end: %w", err)
	}

	return nil
}

// SignRequest is the request to sign a file.
type SignRequest struct {
	Cmd      string      `json:"cmd"`
	UserID   string      `json:"userId"`
	Filename string      `json:"filename"`
	Metadata interface{} `json:"metadata,omitempty"`
}

// SignResponse is the response from a sign request.
type SignResponse struct {
	Status             string `json:"status"`
	UserID             string `json:"userId"`
	Filename           string `json:"filename"`
	HashAlgorithm      string `json:"hashAlgorithm"`
	HashHex            string `json:"hashHex"`
	SignatureAlgorithm string `json:"signatureAlgorithm"`
	SignatureBase64    string `json:"signatureBase64"`
	Error              string `json:"error,omitempty"`
	Message            string `json:"message,omitempty"`
	Hint               string `json:"hint,omitempty"`
}

// Sign signs a previously uploaded file with a user's key.
func (c *Client) Sign(ctx context.Context, userID string, filename string, metadata interface{}) (*SignResponse, error) {
	req := SignRequest{
		Cmd:      "sign",
		UserID:   userID,
		Filename: filename,
		Metadata: metadata,
	}
	resp := &SignResponse{}
	err := c.sendRequest(ctx, req, resp)
	if err != nil {
		return nil, err
	}
	if resp.Status != "ok" {
		return nil, fmt.Errorf("sign failed: %s - %s", resp.Error, resp.Message)
	}
	return resp, nil
}

// SignFile is a convenience method that uploads a file and signs it in one call.
func (c *Client) SignFile(ctx context.Context, userID string, filename string, data io.Reader, metadata interface{}) (*SignResponse, error) {
	if err := c.UploadFile(ctx, filename, data); err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	return c.Sign(ctx, userID, filename, metadata)
}

// sendRequest sends a JSON request to the HSM and unmarshals the response.
func (c *Client) sendRequest(ctx context.Context, reqObj interface{}, respObj interface{}) error {
	reqBody, err := json.Marshal(reqObj)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/cmd", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.bearerToken != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.bearerToken))
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if err := json.Unmarshal(respBody, respObj); err != nil {
		return fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBody))
	}

	return nil
}
