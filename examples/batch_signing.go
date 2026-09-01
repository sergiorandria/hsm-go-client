package main

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/sergiorandria/hsm-go-client/hsm"
)

// Example: Batch signing multiple documents
func main() {
	client := hsm.NewClient(hsm.Config{
		BaseURL:     "http://192.168.0.102",
		BearerToken: "your-token-here",
	})

	ctx := context.Background()

	// Create keys for multiple users
	users := []string{"alice", "bob", "charlie"}
	for _, user := range users {
		fmt.Printf("Creating key for %s...\n", user)
		if _, err := client.CreateKey(ctx, user, false); err != nil {
			// Ignore if key already exists
			fmt.Printf("  (skipped: key may already exist)\n")
		}
	}

	// Documents to sign
	documents := map[string][]byte{
		"contract.txt": []byte("This is a contract"),
		"invoice.txt":  []byte("Invoice #12345"),
		"receipt.txt":  []byte("Receipt from store"),
	}

	// Sign each document with each user concurrently
	var wg sync.WaitGroup
	results := make(chan SignResult, len(users)*len(documents))

	for _, user := range users {
		for filename, content := range documents {
			wg.Add(1)
			go func(u, f string, c []byte) {
				defer wg.Done()
				signDocument(client, ctx, u, f, c, results)
			}(user, filename, content)
		}
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for result := range results {
		if result.Err != nil {
			fmt.Printf("✗ %s (%s): %v\n", result.Filename, result.UserID, result.Err)
		} else {
			fmt.Printf("✓ %s (%s): hash=%s\n", result.Filename, result.UserID, result.Hash)
		}
	}
}

type SignResult struct {
	UserID   string
	Filename string
	Hash     string
	Err      error
}

func signDocument(client *hsm.Client, ctx context.Context, userID, filename string, content []byte, results chan<- SignResult) {
	// Upload and sign
	data := bytes.NewReader(content)
	signResp, err := client.SignFile(ctx, userID, filename, data, map[string]string{
		"user": userID,
	})

	var hash string
	if err == nil {
		hash = signResp.HashHex
	}

	results <- SignResult{
		UserID:   userID,
		Filename: filename,
		Hash:     hash,
		Err:      err,
	}
}
