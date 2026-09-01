// Package esp32 is deprecated: use github.com/sergiorandria/hsm-go-client/hsm/http for
// generic microcontroller HSMs (ESP32, Raspberry Pi, self-made boards).
// This package is kept for backward compatibility and re-exports the HTTP backend.
package esp32

import (
	hhttp "github.com/sergiorandria/hsm-go-client/hsm/http"
	"net/http"
	"time"
)

// Deprecated: use hhttp.DefaultChunkSize
const DefaultChunkSize = hhttp.DefaultChunkSize

// Deprecated: use http.Client
type Client = hhttp.Client

// Deprecated: use http.Config
type Config struct {
	BaseURL     string
	BearerToken string
	HTTPClient  *http.Client
	ChunkSize   int
	Timeout     time.Duration
}

// Deprecated: use http.NewClient
func NewClient(cfg Config) *Client {
	return hhttp.NewClient(hhttp.Config{
		BaseURL:     cfg.BaseURL,
		BearerToken: cfg.BearerToken,
		HTTPClient:  cfg.HTTPClient,
		ChunkSize:   cfg.ChunkSize,
		Timeout:     cfg.Timeout,
	})
}

type CreateKeyRequest = hhttp.CreateKeyRequest
type CreateKeyResponse = hhttp.CreateKeyResponse
type UploadStartRequest = hhttp.UploadStartRequest
type UploadStartResponse = hhttp.UploadStartResponse
type UploadChunkRequest = hhttp.UploadChunkRequest
type UploadChunkResponse = hhttp.UploadChunkResponse
type UploadEndRequest = hhttp.UploadEndRequest
type UploadEndResponse = hhttp.UploadEndResponse
type SignRequest = hhttp.SignRequest
type SignResponse = hhttp.SignResponse
