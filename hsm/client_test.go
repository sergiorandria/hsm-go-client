package hsm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientShim(t *testing.T) {
	client := NewClient(Config{BaseURL: "http://localhost:8080"})
	require.NotNil(t, client)
}

func TestNewClientDefaultsShim(t *testing.T) {
	client := NewClient(Config{})
	require.NotNil(t, client)
}

func TestCreateKeyShim(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","userId":"user123","publicKeyAlgorithm":"ECDSA-P256","publicKeyPem":"-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE...\n-----END PUBLIC KEY-----"}`))
	}))
	defer server.Close()
	client := NewClient(Config{BaseURL: server.URL, BearerToken: "test-token"})
	resp, err := client.CreateKey(context.Background(), "user123", false)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}

func TestSignShim(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","userId":"user123","filename":"test.txt","hashAlgorithm":"SHA-256","hashHex":"abc123","signatureAlgorithm":"ECDSA-SHA256","signatureBase64":"MEUCIQDx..."}`))
	}))
	defer server.Close()
	client := NewClient(Config{BaseURL: server.URL})
	resp, err := client.Sign(context.Background(), "user123", "test.txt", nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}
