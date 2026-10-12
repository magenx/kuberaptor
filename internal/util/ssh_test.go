// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package util

import "testing"

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
