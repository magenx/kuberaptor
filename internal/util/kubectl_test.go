package util

import (
	"testing"
)

func TestKubectlClient_ResourceExists(t *testing.T) {
	// This is a basic unit test that verifies the ResourceExists method can be called
	// In a real scenario, this would require a running Kubernetes cluster or a mock
	// For now, we just verify the method exists and can be called

	client := NewKubectlClient("/tmp/test-kubeconfig")

	// Test with non-existent resource (should return false since kubeconfig doesn't exist)
	exists := client.ResourceExists("deployment", "test", "default")
	if exists {
		t.Error("Expected ResourceExists to return false for non-existent kubeconfig")
	}
}

func TestNewKubectlClient(t *testing.T) {
	// Test that NewKubectlClient creates a client
	client := NewKubectlClient("/tmp/test-kubeconfig")
	if client == nil {
		t.Error("Expected NewKubectlClient to return a non-nil client")
	}

	if client.kubeconfigPath == "" {
		t.Error("Expected kubeconfigPath to be set")
	}
}
