// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package util

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gossh "golang.org/x/crypto/ssh"
)

// makeTestPublicKeyBytes generates a valid SSH public key for testing
func makeTestPublicKeyBytes(t *testing.T) []byte {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	sshPub, err := gossh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("ssh.NewPublicKey: %v", err)
	}
	return gossh.MarshalAuthorizedKey(sshPub)
}

func TestNewSSHFromKeys(t *testing.T) {
	privateKey := []byte("private-key-content")
	publicKey := []byte("public-key-content")

	client := NewSSHFromKeys(privateKey, publicKey)

	if client == nil {
		t.Fatal("NewSSHFromKeys returned nil")
	}

	if string(client.privateKey) != string(privateKey) {
		t.Errorf("expected private key %q, got %q", privateKey, client.privateKey)
	}

	if string(client.publicKey) != string(publicKey) {
		t.Errorf("expected public key %q, got %q", publicKey, client.publicKey)
	}
}

func TestNewSSHFromKeys_EmptyKeys(t *testing.T) {
	client := NewSSHFromKeys(nil, nil)
	if client == nil {
		t.Fatal("NewSSHFromKeys returned nil with nil keys")
	}
	if len(client.privateKey) != 0 {
		t.Errorf("expected empty private key, got %d bytes", len(client.privateKey))
	}
	if len(client.publicKey) != 0 {
		t.Errorf("expected empty public key, got %d bytes", len(client.publicKey))
	}
}

func TestGetPublicKey(t *testing.T) {
	t.Run("returns public key when set", func(t *testing.T) {
		expected := []byte("ssh-rsa AAAA... test@host")
		client := NewSSHFromKeys([]byte("private"), expected)

		key, err := client.GetPublicKey()
		if err != nil {
			t.Fatalf("GetPublicKey() error = %v", err)
		}
		if string(key) != string(expected) {
			t.Errorf("GetPublicKey() = %s, want %s", key, expected)
		}
	})

	t.Run("returns error when no public key set", func(t *testing.T) {
		client := NewSSHFromKeys([]byte("private"), nil)

		_, err := client.GetPublicKey()
		if err == nil {
			t.Error("GetPublicKey() expected error for nil public key, got nil")
		}
	})

	t.Run("returns error when empty public key", func(t *testing.T) {
		client := NewSSHFromKeys([]byte("private"), []byte{})

		_, err := client.GetPublicKey()
		if err == nil {
			t.Error("GetPublicKey() expected error for empty public key, got nil")
		}
	})
}

func TestSetBastion(t *testing.T) {
	client := NewSSHFromKeys([]byte("private"), []byte("public"))

	// Initially no bastion
	if client.bastionHost != "" {
		t.Errorf("expected empty bastion host initially, got %q", client.bastionHost)
	}
	if client.bastionPort != 0 {
		t.Errorf("expected zero bastion port initially, got %d", client.bastionPort)
	}

	// Set bastion
	client.SetBastion("1.2.3.4", 22)

	if client.bastionHost != "1.2.3.4" {
		t.Errorf("expected bastionHost '1.2.3.4', got %q", client.bastionHost)
	}
	if client.bastionPort != 22 {
		t.Errorf("expected bastionPort 22, got %d", client.bastionPort)
	}

	// Update bastion
	client.SetBastion("5.6.7.8", 2222)

	if client.bastionHost != "5.6.7.8" {
		t.Errorf("expected bastionHost '5.6.7.8', got %q", client.bastionHost)
	}
	if client.bastionPort != 2222 {
		t.Errorf("expected bastionPort 2222, got %d", client.bastionPort)
	}
}

func TestCalculateFingerprint(t *testing.T) {
	// Generate a valid SSH public key for testing
	validKey := makeTestPublicKeyBytes(t)

	tests := []struct {
		name        string
		publicKey   []byte
		expectError bool
		checkFormat bool
	}{
		{
			name:        "valid ed25519 public key",
			publicKey:   validKey,
			expectError: false,
			checkFormat: true,
		},
		{
			name:        "invalid key format - no space",
			publicKey:   []byte("invalidssh"),
			expectError: true,
		},
		{
			name:        "invalid base64 data",
			publicKey:   []byte("ssh-rsa not!valid!base64==="),
			expectError: true,
		},
		{
			name:        "empty key",
			publicKey:   []byte(""),
			expectError: true,
		},
		{
			name:        "only whitespace",
			publicKey:   []byte("   "),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fingerprint, err := CalculateFingerprint(tt.publicKey)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil (fingerprint = %q)", fingerprint)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.checkFormat {
				// Fingerprint should be colon-separated hex pairs: xx:xx:xx:...
				parts := strings.Split(fingerprint, ":")
				if len(parts) != 16 {
					t.Errorf("expected 16 colon-separated hex pairs, got %d: %q", len(parts), fingerprint)
				}
				for _, part := range parts {
					if len(part) != 2 {
						t.Errorf("expected 2-char hex part, got %q in fingerprint %q", part, fingerprint)
					}
				}
			}
		})
	}
}

func TestCalculateFingerprintFromPath(t *testing.T) {
	validKey := makeTestPublicKeyBytes(t)

	t.Run("valid key file", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "test.pub")
		if err := os.WriteFile(keyPath, validKey, 0600); err != nil {
			t.Fatalf("failed to write key file: %v", err)
		}

		fingerprint, err := CalculateFingerprintFromPath(keyPath)
		if err != nil {
			t.Fatalf("CalculateFingerprintFromPath() error = %v", err)
		}
		if fingerprint == "" {
			t.Error("CalculateFingerprintFromPath() returned empty fingerprint")
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := CalculateFingerprintFromPath("/nonexistent/path/key.pub")
		if err == nil {
			t.Error("expected error for non-existent file, got nil")
		}
	})

	t.Run("fingerprint matches direct calculation", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "test.pub")
		if err := os.WriteFile(keyPath, validKey, 0600); err != nil {
			t.Fatalf("failed to write key file: %v", err)
		}

		fpFromPath, err := CalculateFingerprintFromPath(keyPath)
		if err != nil {
			t.Fatalf("CalculateFingerprintFromPath() error = %v", err)
		}

		fpDirect, err := CalculateFingerprint(validKey)
		if err != nil {
			t.Fatalf("CalculateFingerprint() error = %v", err)
		}

		if fpFromPath != fpDirect {
			t.Errorf("fingerprints differ: from path = %q, direct = %q", fpFromPath, fpDirect)
		}
	})
}
