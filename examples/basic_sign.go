package main

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"github.com/sergiorandria/hsm-go-client/hsm"
)

// Example: Basic signing workflow
func main() {
	// Create a client connected to the ESP32 HSM
	client := hsm.NewClient(hsm.Config{
		BaseURL:     "http://192.168.0.102",
		BearerToken: "your-token-here", // Set to "" if auth is disabled
	})

	ctx := context.Background()

	// Step 1: Create a key for a user (or skip if key already exists)
	fmt.Println("Creating key for user 'alice'...")
	keyResp, err := client.CreateKey(ctx, "alice", false)
	if err != nil {
		log.Fatalf("Failed to create key: %v", err)
	}
	fmt.Printf("✓ Key created. Algorithm: %s\n", keyResp.PublicKeyAlgorithm)
	fmt.Printf("  Public Key:\n%s\n", keyResp.PublicKeyPEM)

	// Step 2: Upload a file for signing
	fileData := bytes.NewReader([]byte("This is the document to sign"))
	filename := "document.txt"

	fmt.Printf("\nUploading file '%s'...\n", filename)
	if err := client.UploadFile(ctx, filename, fileData); err != nil {
		log.Fatalf("Failed to upload file: %v", err)
	}
	fmt.Println("✓ File uploaded")

	// Step 3: Sign the file
	fmt.Println("\nSigning file...")
	signResp, err := client.Sign(ctx, "alice", filename, map[string]string{
		"document_type": "contract",
		"version":       "1.0",
	})
	if err != nil {
		log.Fatalf("Failed to sign: %v", err)
	}

	fmt.Println("✓ File signed successfully")
	fmt.Printf("  Hash Algorithm: %s\n", signResp.HashAlgorithm)
	fmt.Printf("  Hash: %s\n", signResp.HashHex)
	fmt.Printf("  Signature Algorithm: %s\n", signResp.SignatureAlgorithm)
	fmt.Printf("  Signature (Base64): %s\n", signResp.SignatureBase64)

	// Step 4: Verify the signature offline (using openssl or similar)
	fmt.Println("\nTo verify the signature offline:")
	fmt.Printf("  1. Save the public key to file: public.pem\n")
	fmt.Printf("  2. Save the file content to: document.txt\n")
	fmt.Printf("  3. Decode the signature from base64\n")
	fmt.Printf("  4. Run: openssl dgst -sha256 -verify public.pem -signature sig.der document.txt\n")
}
