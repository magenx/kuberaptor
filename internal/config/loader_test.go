package config

import (
	"fmt"
	"os"
	"testing"
)

func TestValidateFullConfiguration(t *testing.T) {
	// Create temporary test config file
	configContent := `hetzner_token: test_token_12345678901234567890123456789012
cluster_name: test-cluster
kubeconfig_path: "./kubeconfig"
k3s_version: v1.32.0+k3s1

networking:
  ssh:
    port: 22
    use_agent: false
    public_key_path: "%s"
    private_key_path: "%s"

  allowed_networks:
    ssh:
      - 0.0.0.0/0
    api:
      - 0.0.0.0/0

  public_network:
    ipv4:
      enabled: true
    ipv6:
      enabled: true

  private_network:
    enabled: true
    subnet: 10.0.0.0/16

  cni:
    enabled: true
    mode: flannel

datastore:
  mode: etcd

image: ubuntu-24.04

masters_pool:
  instance_type: cpx22
  instance_count: 1
  locations:
    - nbg1

worker_node_pools:
- name: mariadb
  instance_type: cpx32
  instance_count: 1
  location: nbg1

- name: php
  instance_type: cpx32
  location: nbg1
  autoscaling:
    enabled: true
    min_instances: 1
    max_instances: 3
`

	// Create temporary SSH key files
	tmpDir := t.TempDir()
	pubKeyPath := tmpDir + "/id_ed25519.pub"
	privKeyPath := tmpDir + "/id_ed25519"

	if err := os.WriteFile(pubKeyPath, []byte("ssh-ed25519 AAAA test@test"), 0600); err != nil {
		t.Fatalf("Failed to create temp public key: %v", err)
	}
	if err := os.WriteFile(privKeyPath, []byte("-----BEGIN PRIVATE KEY-----"), 0600); err != nil {
		t.Fatalf("Failed to create temp private key: %v", err)
	}

	// Create config file with actual SSH key paths
	configFile := tmpDir + "/config.yaml"
	finalConfig := []byte(fmt.Sprintf(configContent, pubKeyPath, privKeyPath))

	if err := os.WriteFile(configFile, finalConfig, 0644); err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}

	// Load and validate configuration
	loader, err := NewLoader(configFile, "", true)
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	if err := loader.Validate("create"); err != nil {
		t.Fatalf("Basic validation failed: %v", err)
	}

	validator := NewValidator(loader.Settings)
	err = validator.Validate()

	// Should succeed with valid autoscaling configuration
	if err != nil {
		t.Errorf("Expected validation to succeed, got error: %v", err)
	}

	// Verify the autoscaling pool was parsed correctly
	if len(loader.Settings.WorkerNodePools) != 2 {
		t.Fatalf("Expected 2 worker pools, got %d", len(loader.Settings.WorkerNodePools))
	}

	// Check the php pool
	phpPool := loader.Settings.WorkerNodePools[1]
	if phpPool.Name == nil || *phpPool.Name != "php" {
		t.Error("Expected second pool to be named 'php'")
	}

	if !phpPool.AutoscalingEnabled() {
		t.Error("Expected php pool to have autoscaling enabled")
	}

	if phpPool.Autoscaling.MinInstances != 1 {
		t.Errorf("Expected min_instances=1, got %d", phpPool.Autoscaling.MinInstances)
	}

	if phpPool.Autoscaling.MaxInstances != 3 {
		t.Errorf("Expected max_instances=3, got %d", phpPool.Autoscaling.MaxInstances)
	}

	// Check the mariadb pool (static)
	mariadbPool := loader.Settings.WorkerNodePools[0]
	if mariadbPool.Name == nil || *mariadbPool.Name != "mariadb" {
		t.Error("Expected first pool to be named 'mariadb'")
	}

	if mariadbPool.AutoscalingEnabled() {
		t.Error("Expected mariadb pool to not have autoscaling enabled")
	}

	if mariadbPool.InstanceCount != 1 {
		t.Errorf("Expected instance_count=1, got %d", mariadbPool.InstanceCount)
	}
}

func TestSetDefaults_NodePoolClusterNamePrefix(t *testing.T) {
	// Test that SetDefaults correctly sets IncludeClusterNameAsPrefix on node pools
	configContent := `hetzner_token: test_token
cluster_name: magenx
kubeconfig_path: "./kubeconfig"
k3s_version: v1.32.0+k3s1

masters_pool:
  instance_type: cpx22
  instance_count: 1
  locations:
    - nbg1

worker_node_pools:
- name: php
  instance_type: cpx32
  location: nbg1
  autoscaling:
    enabled: true
    min_instances: 1
    max_instances: 3
`

	// Create config file
	tmpDir := t.TempDir()
	configFile := tmpDir + "/config.yaml"
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}

	// Load configuration (which calls SetDefaults)
	loader, err := NewLoader(configFile, "", false)
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Verify that IncludeClusterNameAsPrefix is true after SetDefaults
	if !loader.Settings.MastersPool.IncludeClusterNameAsPrefix {
		t.Error("Expected MastersPool.IncludeClusterNameAsPrefix to be true after SetDefaults")
	}

	if len(loader.Settings.WorkerNodePools) != 1 {
		t.Fatalf("Expected 1 worker pool, got %d", len(loader.Settings.WorkerNodePools))
	}

	phpPool := loader.Settings.WorkerNodePools[0]
	if !phpPool.IncludeClusterNameAsPrefix {
		t.Error("Expected WorkerPool.IncludeClusterNameAsPrefix to be true after SetDefaults")
	}
}
