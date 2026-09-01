package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	cfg := Config{
		BaseURL: "http://localhost:8080",
	}
	client := NewClient(cfg)
	assert.Equal(t, "http://localhost:8080", client.baseURL)
	assert.Equal(t, DefaultChunkSize, client.chunkSize)
}

func TestNewClientDefaults(t *testing.T) {
	client := NewClient(Config{})
	assert.Equal(t, "http://192.168.0.102", client.baseURL)
	assert.Equal(t, DefaultChunkSize, client.chunkSize)
	assert.NotNil(t, client.httpClient)
}

func TestCreateKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"status": "ok",
			"userId": "user123",
			"publicKeyAlgorithm": "ECDSA-P256",
			"publicKeyPem": "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE...\n-----END PUBLIC KEY-----"
		}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:     server.URL,
		BearerToken: "test-token",
	})

	resp, err := client.CreateKey(context.Background(), "user123", false)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "user123", resp.UserID)
	assert.Equal(t, "ECDSA-P256", resp.PublicKeyAlgorithm)
	assert.Contains(t, resp.PublicKeyPEM, "-----BEGIN PUBLIC KEY-----")
}

func TestUploadFile(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch callCount {
		case 1: // uploadStart
			w.Write([]byte(`{"status": "ok", "filename": "test.txt"}`))
		case 2: // uploadChunk
			w.Write([]byte(`{"status": "ok", "filename": "test.txt", "bytesWritten": 100, "totalBytes": 100}`))
		case 3: // uploadEnd
			w.Write([]byte(`{"status": "ok", "filename": "test.txt", "totalBytes": 100}`))
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	data := bytes.NewReader([]byte("test data"))

	err := client.UploadFile(context.Background(), "test.txt", data)
	require.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestSign(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"status": "ok",
			"userId": "user123",
			"filename": "test.txt",
			"hashAlgorithm": "SHA-256",
			"hashHex": "abc123",
			"signatureAlgorithm": "ECDSA-SHA256",
			"signatureBase64": "MEUCIQDx..."
		}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})

	resp, err := client.Sign(context.Background(), "user123", "test.txt", nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "user123", resp.UserID)
	assert.Equal(t, "test.txt", resp.Filename)
	assert.Equal(t, "SHA-256", resp.HashAlgorithm)
}

func TestSignFile(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch callCount {
		case 1: // uploadStart
			w.Write([]byte(`{"status": "ok", "filename": "test.txt"}`))
		case 2: // uploadChunk
			w.Write([]byte(`{"status": "ok", "filename": "test.txt", "bytesWritten": 9, "totalBytes": 9}`))
		case 3: // uploadEnd
			w.Write([]byte(`{"status": "ok", "filename": "test.txt", "totalBytes": 9}`))
		case 4: // sign
			w.Write([]byte(`{
				"status": "ok",
				"userId": "user123",
				"filename": "test.txt",
				"hashAlgorithm": "SHA-256",
				"hashHex": "abc123",
				"signatureAlgorithm": "ECDSA-SHA256",
				"signatureBase64": "MEUCIQDx..."
			}`))
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	data := bytes.NewReader([]byte("test data"))

	resp, err := client.SignFile(context.Background(), "user123", "test.txt", data, nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, 4, callCount)
}

func TestErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{
			"status": "error",
			"error": "key_already_exists",
			"message": "A key for this user already exists",
			"hint": "Set overwrite:true to replace the existing key"
		}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})
	resp, err := client.CreateKey(context.Background(), "user123", false)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "key_already_exists")
}

func TestContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		Timeout: 50 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.CreateKey(ctx, "user123", false)
	assert.Error(t, err)
}

func TestUploadFileLargeData(t *testing.T) {
	chunkCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		var response string
		if chunkCount == 0 {
			response = `{"status": "ok", "filename": "large.bin"}`
		} else if chunkCount == 1 || chunkCount == 2 {
			chunkCount++
			response = `{"status": "ok", "filename": "large.bin", "bytesWritten": 10000, "totalBytes": 10000}`
		} else {
			response = `{"status": "ok", "filename": "large.bin", "totalBytes": 20000}`
		}
		chunkCount++
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:   server.URL,
		ChunkSize: 10000,
	})

	data := bytes.NewReader(make([]byte, 20000))
	err := client.UploadFile(context.Background(), "large.bin", data)
	require.NoError(t, err)
}

func BenchmarkSign(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"status": "ok",
			"userId": "user123",
			"filename": "test.txt",
			"hashAlgorithm": "SHA-256",
			"hashHex": "abc123",
			"signatureAlgorithm": "ECDSA-SHA256",
			"signatureBase64": "MEUCIQDx..."
		}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.Sign(context.Background(), "user123", "test.txt", nil)
	}
}
