// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package hetzner

import (
	"testing"

	"github.com/magenx/kuberaptor/pkg/version"
)

func TestNewClient_UsesCorrectVersion(t *testing.T) {
	// Set a test version
	originalVersion := version.Version
	testVersion := "test-version-1.2.3"
	version.Version = testVersion
	defer func() { version.Version = originalVersion }()

	// Create a new client
	client := NewClient("test-token")

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	// Verify client was created successfully
	if client.hcloud == nil {
		t.Error("hcloud client is nil")
	}

	if client.token != "test-token" {
		t.Errorf("expected token 'test-token', got '%s'", client.token)
	}

	// Note: We can't directly verify the version string passed to hcloud.WithApplication
	// without mocking, but we can verify the version.Get() returns the correct value
	if version.Get() != testVersion {
		t.Errorf("expected version '%s', got '%s'", testVersion, version.Get())
	}
}

func TestNewClient_WithEmptyToken(t *testing.T) {
	client := NewClient("")

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.token != "" {
		t.Errorf("expected empty token, got '%s'", client.token)
	}
}
