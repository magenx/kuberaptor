// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package config

import (
	"strings"
	"testing"
)

func newAutoscalerValidatorConfig(pools ...WorkerNodePool) *Main {
	cfg := &Main{
		ClusterName: "test-cluster",
		Networking: Networking{
			PrivateNetwork: PrivateNetwork{Enabled: true, Subnet: "10.0.0.0/16"},
		},
		Addons: Addons{
			ClusterAutoscaler: &ClusterAutoscaler{Enabled: true, ContainerImageTag: "v1.37.0"},
		},
		WorkerNodePools: pools,
	}
	return cfg
}

func autoscalingTestPool(name string, maxInstances int, autoscaling Autoscaling) WorkerNodePool {
	autoscaling.Enabled = true
	autoscaling.MaxInstances = maxInstances
	return WorkerNodePool{
		NodePool: NodePool{
			Name:         &name,
			InstanceType: "cpx32",
			Autoscaling:  &autoscaling,
		},
		Locations: []string{"fsn1"},
	}
}

func runAutoscalerValidation(cfg *Main) []string {
	v := NewValidator(cfg)
	v.validateClusterAutoscaler()
	return v.errors
}

func TestClusterAutoscalerMinorVersion(t *testing.T) {
	tests := []struct {
		tag   string
		minor int
		ok    bool
	}{
		{"v1.37.0", 37, true},
		{"1.34.2", 34, true},
		{"v1.36.0-rc.1", 36, true},
		{"latest", 0, false},
		{"custom-build", 0, false},
		{"", 0, false},
	}
	for _, tt := range tests {
		ca := &ClusterAutoscaler{ContainerImageTag: tt.tag}
		minor, ok := ca.ClusterAutoscalerMinorVersion()
		if minor != tt.minor || ok != tt.ok {
			t.Errorf("tag %q: got (%d, %t), expected (%d, %t)", tt.tag, minor, ok, tt.minor, tt.ok)
		}
	}
}

func TestValidateClusterAutoscaler(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(cfg *Main)
		pools    []WorkerNodePool
		expected []string // substrings; empty means no errors
	}{
		{
			name:  "no optional settings",
			pools: []WorkerNodePool{autoscalingTestPool("workers", 20, Autoscaling{})},
		},
		{
			name: "all settings valid",
			mutate: func(cfg *Main) {
				cfg.Addons.ClusterAutoscaler.DefaultSubnetIPRange = "10.0.0.0/16"
			},
			pools: []WorkerNodePool{autoscalingTestPool("workers", 5, Autoscaling{
				Image:          "ubuntu-24.04",
				SubnetIPRange:  "10.0.0.0/16",
				Firewalls:      []string{"extra"},
				PlacementGroup: "test-cluster-workers",
			})},
		},
		{
			name:     "image requires 1.34",
			mutate:   func(cfg *Main) { cfg.Addons.ClusterAutoscaler.ContainerImageTag = "v1.33.0" },
			pools:    []WorkerNodePool{autoscalingTestPool("workers", 3, Autoscaling{Image: "ubuntu-24.04"})},
			expected: []string{"autoscaling.image requires cluster autoscaler v1.34.0"},
		},
		{
			name:     "firewalls require 1.37",
			mutate:   func(cfg *Main) { cfg.Addons.ClusterAutoscaler.ContainerImageTag = "v1.36.2" },
			pools:    []WorkerNodePool{autoscalingTestPool("workers", 3, Autoscaling{Firewalls: []string{"extra"}})},
			expected: []string{"autoscaling.firewalls requires cluster autoscaler v1.37.0"},
		},
		{
			name:   "custom image tag skips version check",
			mutate: func(cfg *Main) { cfg.Addons.ClusterAutoscaler.ContainerImageTag = "my-build" },
			pools:  []WorkerNodePool{autoscalingTestPool("workers", 3, Autoscaling{Firewalls: []string{"extra"}})},
		},
		{
			name:     "empty firewall",
			pools:    []WorkerNodePool{autoscalingTestPool("workers", 3, Autoscaling{Firewalls: []string{" "}})},
			expected: []string{"autoscaling.firewalls[0] is empty"},
		},
		{
			name:     "invalid subnet",
			pools:    []WorkerNodePool{autoscalingTestPool("workers", 3, Autoscaling{SubnetIPRange: "10.0.0.0"})},
			expected: []string{"invalid CIDR"},
		},
		{
			name:     "subnet not a network address",
			pools:    []WorkerNodePool{autoscalingTestPool("workers", 3, Autoscaling{SubnetIPRange: "10.0.0.1/16"})},
			expected: []string{"is not a network address"},
		},
		{
			name:     "subnet not in created network",
			pools:    []WorkerNodePool{autoscalingTestPool("workers", 3, Autoscaling{SubnetIPRange: "10.0.16.0/20"})},
			expected: []string{"is not a subnet of the private network"},
		},
		{
			name: "subnet in existing network is not checked",
			mutate: func(cfg *Main) {
				cfg.Networking.PrivateNetwork.ExistingNetworkName = "my-network"
			},
			pools: []WorkerNodePool{autoscalingTestPool("workers", 3, Autoscaling{SubnetIPRange: "10.0.16.0/20"})},
		},
		{
			name:     "subnet requires private network",
			mutate:   func(cfg *Main) { cfg.Networking.PrivateNetwork.Enabled = false },
			pools:    []WorkerNodePool{autoscalingTestPool("workers", 3, Autoscaling{SubnetIPRange: "10.0.0.0/16"})},
			expected: []string{"requires networking.private_network.enabled"},
		},
		{
			name: "default subnet validated",
			mutate: func(cfg *Main) {
				cfg.Addons.ClusterAutoscaler.DefaultSubnetIPRange = "192.168.0.0/24"
				cfg.Addons.ClusterAutoscaler.ContainerImageTag = "v1.34.0"
			},
			pools: []WorkerNodePool{autoscalingTestPool("workers", 3, Autoscaling{})},
			expected: []string{
				"default_subnet_ip_range requires cluster autoscaler v1.35.0",
				"is not a subnet of the private network",
			},
		},
		{
			name: "placement group over limit across pools",
			pools: []WorkerNodePool{
				autoscalingTestPool("a", 6, Autoscaling{PlacementGroup: "shared"}),
				autoscalingTestPool("b", 5, Autoscaling{PlacementGroup: "shared"}),
			},
			expected: []string{"placement group shared: autoscaling max_instances plus static nodes is 11"},
		},
		{
			name: "placement group counts static nodes",
			pools: []WorkerNodePool{
				{
					NodePool: NodePool{
						Name:           stringPointer("static"),
						InstanceType:   "cpx22",
						InstanceCount:  4,
						PlacementGroup: &PlacementGroupConfig{Name: "web", Type: "spread"},
					},
					Locations: []string{"fsn1"},
				},
				autoscalingTestPool("web", 7, Autoscaling{PlacementGroup: "test-cluster-web"}),
			},
			expected: []string{"placement group test-cluster-web: autoscaling max_instances plus static nodes is 11"},
		},
		{
			name:   "autoscaler disabled skips checks",
			mutate: func(cfg *Main) { cfg.Addons.ClusterAutoscaler.Enabled = false },
			pools:  []WorkerNodePool{autoscalingTestPool("workers", 3, Autoscaling{SubnetIPRange: "bad"})},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newAutoscalerValidatorConfig(tt.pools...)
			if tt.mutate != nil {
				tt.mutate(cfg)
			}
			errors := runAutoscalerValidation(cfg)
			if len(tt.expected) == 0 {
				if len(errors) > 0 {
					t.Errorf("expected no errors, got %v", errors)
				}
				return
			}
			if len(errors) != len(tt.expected) {
				t.Errorf("expected %d errors, got %v", len(tt.expected), errors)
			}
			for _, want := range tt.expected {
				found := false
				for _, got := range errors {
					if strings.Contains(got, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, got %v", want, errors)
				}
			}
		})
	}
}

func stringPointer(s string) *string {
	return &s
}
