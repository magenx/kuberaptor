// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package cluster

import (
	"context"
	"testing"

	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/pkg/hetzner"
)

// TestMasterPoolHetznerLabels verifies that master pool applies custom Hetzner labels
func TestMasterPoolHetznerLabels(t *testing.T) {
	// Create a test configuration with master pool labels
	cfg := &config.Main{
		ClusterName: "test-cluster",
		MastersPool: config.MasterNodePool{
			NodePool: config.NodePool{
				Hetzner: &config.HetznerConfig{
					Labels: []config.Label{
						{Key: "cluster_id", Value: "test-uuid-123"},
						{Key: "environment", Value: "testing"},
						{Key: "managed", Value: "should-not-override"}, // This should be ignored
					},
				},
			},
		},
	}

	// Create a CreatorEnhanced instance
	creator := &CreatorEnhanced{
		Config:        cfg,
		HetznerClient: &hetzner.Client{}, // Mock client
		ctx:           context.Background(),
	}

	// Build labels for a master node
	labels := creator.buildMasterHetznerServerLabels("fsn1")

	// Verify default labels are present
	if labels["cluster"] != "test-cluster" {
		t.Errorf("Expected cluster label 'test-cluster', got '%s'", labels["cluster"])
	}
	if labels["role"] != "master" {
		t.Errorf("Expected role label 'master', got '%s'", labels["role"])
	}
	if labels["location"] != "fsn1" {
		t.Errorf("Expected location label 'fsn1', got '%s'", labels["location"])
	}
	if labels["managed"] != "kuberaptor" {
		t.Errorf("Expected managed label 'kuberaptor', got '%s'", labels["managed"])
	}

	// Verify custom labels are merged
	if labels["cluster_id"] != "test-uuid-123" {
		t.Errorf("Expected cluster_id label 'test-uuid-123', got '%s'", labels["cluster_id"])
	}
	if labels["environment"] != "testing" {
		t.Errorf("Expected environment label 'testing', got '%s'", labels["environment"])
	}

	// Verify that 'managed' label cannot be overridden
	if labels["managed"] == "should-not-override" {
		t.Error("Custom labels should not be able to override 'managed' label")
	}
}

// TestNATGatewayHetznerLabels verifies that NAT gateway applies custom Hetzner labels
func TestNATGatewayHetznerLabels(t *testing.T) {
	// Create a test configuration with NAT gateway labels
	cfg := &config.Main{
		ClusterName: "test-cluster",
		Networking: config.Networking{
			PrivateNetwork: config.PrivateNetwork{
				NATGateway: &config.NATGateway{
					Hetzner: &config.HetznerConfig{
						Labels: []config.Label{
							{Key: "cluster_id", Value: "nat-uuid-456"},
							{Key: "role_type", Value: "gateway"},
						},
					},
				},
			},
		},
	}

	// Create a CreatorEnhanced instance
	creator := &CreatorEnhanced{
		Config:        cfg,
		HetznerClient: &hetzner.Client{}, // Mock client
		ctx:           context.Background(),
	}

	// Build labels for a NAT gateway
	labels := creator.buildNATGatewayHetznerServerLabels("nbg1")

	// Verify default labels are present
	if labels["cluster"] != "test-cluster" {
		t.Errorf("Expected cluster label 'test-cluster', got '%s'", labels["cluster"])
	}
	if labels["role"] != "nat-gateway" {
		t.Errorf("Expected role label 'nat-gateway', got '%s'", labels["role"])
	}
	if labels["location"] != "nbg1" {
		t.Errorf("Expected location label 'nbg1', got '%s'", labels["location"])
	}
	if labels["managed"] != "kuberaptor" {
		t.Errorf("Expected managed label 'kuberaptor', got '%s'", labels["managed"])
	}

	// Verify custom labels are merged
	if labels["cluster_id"] != "nat-uuid-456" {
		t.Errorf("Expected cluster_id label 'nat-uuid-456', got '%s'", labels["cluster_id"])
	}
	if labels["role_type"] != "gateway" {
		t.Errorf("Expected role_type label 'gateway', got '%s'", labels["role_type"])
	}
}

// TestNATGatewayHetznerLabelsNil verifies that NAT gateway handles nil Hetzner config gracefully
func TestNATGatewayHetznerLabelsNil(t *testing.T) {
	// Create a test configuration without NAT gateway Hetzner labels
	cfg := &config.Main{
		ClusterName: "test-cluster",
		Networking: config.Networking{
			PrivateNetwork: config.PrivateNetwork{
				NATGateway: &config.NATGateway{
					Hetzner: nil, // No Hetzner config
				},
			},
		},
	}

	// Create a CreatorEnhanced instance
	creator := &CreatorEnhanced{
		Config:        cfg,
		HetznerClient: &hetzner.Client{}, // Mock client
		ctx:           context.Background(),
	}

	// Build labels for a NAT gateway
	labels := creator.buildNATGatewayHetznerServerLabels("nbg1")

	// Verify only default labels are present
	if len(labels) != 4 {
		t.Errorf("Expected 4 labels, got %d", len(labels))
	}
	if labels["cluster"] != "test-cluster" {
		t.Errorf("Expected cluster label 'test-cluster', got '%s'", labels["cluster"])
	}
	if labels["role"] != "nat-gateway" {
		t.Errorf("Expected role label 'nat-gateway', got '%s'", labels["role"])
	}
	if labels["location"] != "nbg1" {
		t.Errorf("Expected location label 'nbg1', got '%s'", labels["location"])
	}
	if labels["managed"] != "kuberaptor" {
		t.Errorf("Expected managed label 'kuberaptor', got '%s'", labels["managed"])
	}
}
