package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sergiorandria/hsm-go-client/hsm"
)

// Example: Key management
func main() {
	client := hsm.NewClient(hsm.Config{
		BaseURL:     "http://192.168.0.102",
		BearerToken: "your-token-here",
	})

	ctx := context.Background()

	// Create keys for different users
	users := []string{"user_a", "user_b", "user_c"}

	fmt.Println("Creating keys...")
	for _, userID := range users {
		resp, err := client.CreateKey(ctx, userID, false)
		if err != nil {
			log.Printf("Failed to create key for %s: %v", userID, err)
			continue
		}

		fmt.Printf("\n✓ Key created for %s\n", userID)
		fmt.Printf("  Algorithm: %s\n", resp.PublicKeyAlgorithm)
		fmt.Printf("  Public Key (first 100 chars):\n  %s...\n", resp.PublicKeyPEM[:100])
	}

	// Try to create a key that already exists
	fmt.Println("\n\nTrying to create duplicate key for 'user_a'...")
	if _, err := client.CreateKey(ctx, "user_a", false); err != nil {
		fmt.Printf("✓ Got expected error: %v\n", err)
	}

	// Try with overwrite flag
	fmt.Println("\nRecreating key for 'user_a' with overwrite=true...")
	if _, err := client.CreateKey(ctx, "user_a", true); err != nil {
		log.Fatalf("Failed: %v", err)
	}
	fmt.Printf("✓ Key recreated successfully\n")
}
