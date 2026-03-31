// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPublicNetworkIPv4_UnmarshalYAML_BooleanFormat(t *testing.T) {
	yamlData := `
public_network:
  ipv4: true
  ipv6: false
`
	var config struct {
		PublicNetwork PublicNetwork `yaml:"public_network"`
	}

	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML with boolean format: %v", err)
	}

	if config.PublicNetwork.IPv4 == nil {
		t.Fatal("Expected IPv4 to be set, got nil")
	}
	if !config.PublicNetwork.IPv4.Enabled {
		t.Error("Expected IPv4.Enabled to be true, got false")
	}

	if config.PublicNetwork.IPv6 == nil {
		t.Fatal("Expected IPv6 to be set, got nil")
	}
	if config.PublicNetwork.IPv6.Enabled {
		t.Error("Expected IPv6.Enabled to be false, got true")
	}
}

func TestPublicNetworkIPv4_UnmarshalYAML_ObjectFormat(t *testing.T) {
	yamlData := `
public_network:
  ipv4:
    enabled: true
  ipv6:
    enabled: false
`
	var config struct {
		PublicNetwork PublicNetwork `yaml:"public_network"`
	}

	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML with object format: %v", err)
	}

	if config.PublicNetwork.IPv4 == nil {
		t.Fatal("Expected IPv4 to be set, got nil")
	}
	if !config.PublicNetwork.IPv4.Enabled {
		t.Error("Expected IPv4.Enabled to be true, got false")
	}

	if config.PublicNetwork.IPv6 == nil {
		t.Fatal("Expected IPv6 to be set, got nil")
	}
	if config.PublicNetwork.IPv6.Enabled {
		t.Error("Expected IPv6.Enabled to be false, got true")
	}
}

func TestPublicNetworkIPv4_UnmarshalYAML_MixedFormats(t *testing.T) {
	// Test that we can have one as boolean and one as object
	yamlData := `
public_network:
  ipv4: true
  ipv6:
    enabled: false
`
	var config struct {
		PublicNetwork PublicNetwork `yaml:"public_network"`
	}

	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML with mixed formats: %v", err)
	}

	if config.PublicNetwork.IPv4 == nil {
		t.Fatal("Expected IPv4 to be set, got nil")
	}
	if !config.PublicNetwork.IPv4.Enabled {
		t.Error("Expected IPv4.Enabled to be true, got false")
	}

	if config.PublicNetwork.IPv6 == nil {
		t.Fatal("Expected IPv6 to be set, got nil")
	}
	if config.PublicNetwork.IPv6.Enabled {
		t.Error("Expected IPv6.Enabled to be false, got true")
	}
}

func TestCilium_HubbleMetrics_UnmarshalArray(t *testing.T) {
	yamlContent := `
cni:
  mode: cilium
  cilium:
    enabled: true
    hubble_metrics:
      - dns
      - drop
      - tcp
      - flow
      - port-distribution
      - icmp
      - http
`

	var networking Networking
	err := yaml.Unmarshal([]byte(yamlContent), &networking)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if networking.CNI.Mode != "cilium" {
		t.Errorf("Expected CNI mode 'cilium', got '%s'", networking.CNI.Mode)
	}

	if networking.CNI.Cilium == nil {
		t.Fatal("Expected Cilium config to be non-nil")
	}

	if !networking.CNI.Cilium.Enabled {
		t.Error("Expected Cilium to be enabled")
	}

	expectedMetrics := []string{"dns", "drop", "tcp", "flow", "port-distribution", "icmp", "http"}
	if len(networking.CNI.Cilium.HubbleMetrics) != len(expectedMetrics) {
		t.Errorf("Expected %d metrics, got %d", len(expectedMetrics), len(networking.CNI.Cilium.HubbleMetrics))
	}

	for i, metric := range expectedMetrics {
		if i >= len(networking.CNI.Cilium.HubbleMetrics) {
			t.Errorf("Missing metric at index %d: %s", i, metric)
			continue
		}
		if networking.CNI.Cilium.HubbleMetrics[i] != metric {
			t.Errorf("Expected metric '%s' at index %d, got '%s'", metric, i, networking.CNI.Cilium.HubbleMetrics[i])
		}
	}
}

func TestCilium_HubbleMetrics_EmptyArray(t *testing.T) {
	yamlContent := `
cni:
  mode: cilium
  cilium:
    enabled: true
    hubble_metrics: []
`

	var networking Networking
	err := yaml.Unmarshal([]byte(yamlContent), &networking)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if networking.CNI.Cilium == nil {
		t.Fatal("Expected Cilium config to be non-nil")
	}

	if len(networking.CNI.Cilium.HubbleMetrics) != 0 {
		t.Errorf("Expected empty metrics array, got %d items", len(networking.CNI.Cilium.HubbleMetrics))
	}
}

func TestCilium_HubbleMetrics_NotProvided(t *testing.T) {
	yamlContent := `
cni:
  mode: cilium
  cilium:
    enabled: true
`

	var networking Networking
	err := yaml.Unmarshal([]byte(yamlContent), &networking)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if networking.CNI.Cilium == nil {
		t.Fatal("Expected Cilium config to be non-nil")
	}

	if networking.CNI.Cilium.HubbleMetrics != nil {
		t.Errorf("Expected nil metrics when not provided, got %v", networking.CNI.Cilium.HubbleMetrics)
	}
}
