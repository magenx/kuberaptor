// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSSHKeys_InlineKeys(t *testing.T) {
	// Test case: Inline SSH keys should be valid
	config := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		Networking: Networking{
			SSH: SSH{
				Port:       22,
				PublicKey:  "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7examplekeywithenoughcharactersforvalidation test@example.com",
				PrivateKey: "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEAexamplekeywithmorethanonehundredcharactersforpropervalidationofthekeylengthvalidator\n-----END RSA PRIVATE KEY-----",
			},
		},
	}

	validator := NewValidator(config)
	validator.validateSSHKeys()

	// Should not have any errors
	if len(validator.GetErrors()) > 0 {
		t.Errorf("Expected no validation errors for inline keys, got: %v", validator.GetErrors())
	}
}

func TestValidateSSHKeys_PathKeys(t *testing.T) {
	// Create temp directory and test keys
	tmpDir := t.TempDir()
	privKeyPath := filepath.Join(tmpDir, "id_rsa")
	pubKeyPath := filepath.Join(tmpDir, "id_rsa.pub")

	// Create test key files
	if err := os.WriteFile(privKeyPath, []byte("test-private-key"), 0600); err != nil {
		t.Fatalf("Failed to create test private key: %v", err)
	}
	if err := os.WriteFile(pubKeyPath, []byte("ssh-rsa test-public-key"), 0644); err != nil {
		t.Fatalf("Failed to create test public key: %v", err)
	}

	config := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		Networking: Networking{
			SSH: SSH{
				Port:           22,
				PublicKeyPath:  pubKeyPath,
				PrivateKeyPath: privKeyPath,
			},
		},
	}

	validator := NewValidator(config)
	validator.validateSSHKeys()

	// Should not have any errors
	if len(validator.GetErrors()) > 0 {
		t.Errorf("Expected no validation errors for path keys, got: %v", validator.GetErrors())
	}
}

func TestValidateSSHKeys_MissingBoth(t *testing.T) {
	// Test case: No public key specified (neither path nor inline)
	config := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		Networking: Networking{
			SSH: SSH{
				Port:       22,
				PrivateKey: "test-private-key",
			},
		},
	}

	validator := NewValidator(config)
	validator.validateSSHKeys()

	// Should have an error about missing public key
	foundError := false
	for _, err := range validator.GetErrors() {
		if strings.Contains(err, "SSH public_key_path or public_key is required") {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Error("Expected error about missing public key")
	}
}

func TestValidateSSHKeys_BothPathAndInline(t *testing.T) {
	// Test case: Both path and inline key specified - should error
	tmpDir := t.TempDir()
	pubKeyPath := filepath.Join(tmpDir, "id_rsa.pub")
	if err := os.WriteFile(pubKeyPath, []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7examplekey test-public-key"), 0644); err != nil {
		t.Fatalf("Failed to create test public key: %v", err)
	}

	config := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		Networking: Networking{
			SSH: SSH{
				Port:          22,
				PublicKeyPath: pubKeyPath,
				PublicKey:     "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7examplekeywithenoughcharacters test@example.com",
				PrivateKey:    "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEAexamplekeywithmorethanonehundredcharactersforpropervalidation\n-----END RSA PRIVATE KEY-----",
			},
		},
	}

	validator := NewValidator(config)
	validator.validateSSHKeys()

	// Should have an error about specifying both
	foundError := false
	for _, err := range validator.GetErrors() {
		if strings.Contains(err, "Cannot specify both SSH public_key_path and public_key") {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Error("Expected error about specifying both path and inline public key")
	}
}

func TestValidateSSHKeys_PathNotFound(t *testing.T) {
	// Test case: Path specified but file doesn't exist
	config := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		Networking: Networking{
			SSH: SSH{
				Port:           22,
				PublicKeyPath:  "/nonexistent/key.pub",
				PrivateKeyPath: "/nonexistent/key",
			},
		},
	}

	validator := NewValidator(config)
	validator.validateSSHKeys()

	// Should have errors about files not found
	errors := validator.GetErrors()
	if len(errors) < 2 {
		t.Errorf("Expected at least 2 errors for missing key files, got %d: %v", len(errors), errors)
	}

	foundPublicError := false
	foundPrivateError := false
	for _, err := range errors {
		if strings.Contains(err, "SSH public key not found") {
			foundPublicError = true
		}
		if strings.Contains(err, "SSH private key not found") {
			foundPrivateError = true
		}
	}

	if !foundPublicError {
		t.Error("Expected error about public key not found")
	}
	if !foundPrivateError {
		t.Error("Expected error about private key not found")
	}
}

func TestValidateSSHKeys_InlineKeyTooShort(t *testing.T) {
	// Test case: Inline key is too short
	config := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		Networking: Networking{
			SSH: SSH{
				Port:       22,
				PublicKey:  "short",    // Less than 50 characters
				PrivateKey: "tooshort", // Less than 100 characters
			},
		},
	}

	validator := NewValidator(config)
	validator.validateSSHKeys()

	// Should have errors about keys being too short
	foundError := false
	for _, err := range validator.GetErrors() {
		if strings.Contains(err, "appears to be too short") {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Error("Expected error about key being too short")
	}
}

func TestSSHGetPrivateKey_FromInline(t *testing.T) {
	// Test case: Get private key from inline content
	testKey := "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEAexamplekeywithmorethanonehundredcharacters\n-----END RSA PRIVATE KEY-----"
	ssh := &SSH{
		PrivateKey: testKey,
	}

	key, err := ssh.GetPrivateKey()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if string(key) != testKey {
		t.Errorf("Expected '%s', got: %s", testKey, string(key))
	}
}

func TestSSHGetPrivateKey_FromPath(t *testing.T) {
	// Test case: Get private key from file path
	tmpDir := t.TempDir()
	privKeyPath := filepath.Join(tmpDir, "id_rsa")

	expectedContent := "test-private-key-from-file"
	if err := os.WriteFile(privKeyPath, []byte(expectedContent), 0600); err != nil {
		t.Fatalf("Failed to create test private key: %v", err)
	}

	ssh := &SSH{
		PrivateKeyPath: privKeyPath,
	}

	key, err := ssh.GetPrivateKey()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if string(key) != expectedContent {
		t.Errorf("Expected '%s', got: %s", expectedContent, string(key))
	}
}

func TestSSHGetPublicKey_FromInline(t *testing.T) {
	// Test case: Get public key from inline content
	testKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7examplekey test-public-key"
	ssh := &SSH{
		PublicKey: testKey,
	}

	key, err := ssh.GetPublicKey()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if string(key) != testKey {
		t.Errorf("Expected '%s', got: %s", testKey, string(key))
	}
}

func TestSSHGetPublicKey_FromPath(t *testing.T) {
	// Test case: Get public key from file path
	tmpDir := t.TempDir()
	pubKeyPath := filepath.Join(tmpDir, "id_rsa.pub")

	expectedContent := "ssh-rsa test-public-key-from-file"
	if err := os.WriteFile(pubKeyPath, []byte(expectedContent), 0644); err != nil {
		t.Fatalf("Failed to create test public key: %v", err)
	}

	ssh := &SSH{
		PublicKeyPath: pubKeyPath,
	}

	key, err := ssh.GetPublicKey()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if string(key) != expectedContent {
		t.Errorf("Expected '%s', got: %s", expectedContent, string(key))
	}
}

func TestSSHGetPrivateKey_NoneConfigured(t *testing.T) {
	// Test case: No private key configured
	ssh := &SSH{}

	_, err := ssh.GetPrivateKey()
	if err == nil {
		t.Error("Expected error when no private key is configured")
	}

	if !strings.Contains(err.Error(), "no private key configured") {
		t.Errorf("Expected 'no private key configured' error, got: %v", err)
	}
}
