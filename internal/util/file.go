package util

import (
	"fmt"
	"os"
	"path/filepath"
)

// ReadPublicKey reads a public SSH key from a file
func ReadPublicKey(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read public key: %w", err)
	}
	return string(data), nil
}

// ReadPrivateKey reads a private SSH key from a file
func ReadPrivateKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}
	return data, nil
}

// WriteToFile writes data to a file, creating parent directories if needed
func WriteToFile(path string, data []byte, perm os.FileMode) error {
	// Create parent directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}
