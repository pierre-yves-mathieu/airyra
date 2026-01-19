package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

const (
	// Prefix for all generated IDs.
	Prefix = "ar"
	// IDLength is the number of hex characters after the prefix.
	IDLength = 4
)

// Generate creates a new unique ID in the format "ar-xxxx".
func Generate() (string, error) {
	bytes := make([]byte, IDLength/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate ID: %w", err)
	}
	return fmt.Sprintf("%s-%s", Prefix, hex.EncodeToString(bytes)), nil
}

// MustGenerate creates a new unique ID, panicking on error.
func MustGenerate() string {
	id, err := Generate()
	if err != nil {
		panic(err)
	}
	return id
}
