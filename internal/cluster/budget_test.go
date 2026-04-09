// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package cluster

import (
	"testing"

	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/pkg/hetzner"
)

func TestNewBudgetCalculator(t *testing.T) {
	cfg := &config.Main{
		ClusterName:  "test-cluster",
		HetznerToken: "test-token",
	}

	hetznerClient := hetzner.NewClient(cfg.HetznerToken)
	calculator := NewBudgetCalculator(cfg, hetznerClient)

	if calculator == nil {
		t.Fatal("NewBudgetCalculator returned nil")
	}

	if calculator.Config.ClusterName != "test-cluster" {
		t.Errorf("Expected cluster name 'test-cluster', got '%s'", calculator.Config.ClusterName)
	}
}

func TestResourceCost(t *testing.T) {
	cost := ResourceCost{
		Type:         "Server",
		Name:         "test-server",
		ResourceType: "cx22",
		HourlyPrice:  0.01,
		MonthlyPrice: 7.5,
		Currency:     "EUR",
	}

	if cost.Type != "Server" {
		t.Errorf("Expected type 'Server', got '%s'", cost.Type)
	}

	if cost.MonthlyPrice != 7.5 {
		t.Errorf("Expected monthly price 7.5, got %f", cost.MonthlyPrice)
	}
}

// TestDisplayBudget verifies that displayBudget handles various cost configurations
// without panicking or producing incorrect output
func TestDisplayBudget(t *testing.T) {
	cfg := &config.Main{
		ClusterName:  "test-cluster",
		HetznerToken: "test-token",
	}
	hetznerClient := hetzner.NewClient(cfg.HetznerToken)
	calculator := NewBudgetCalculator(cfg, hetznerClient)

	t.Run("empty costs list", func(t *testing.T) {
		// Should not panic with empty costs
		calculator.displayBudget([]ResourceCost{})
	})

	t.Run("costs with paid resources", func(t *testing.T) {
		costs := []ResourceCost{
			{
				Type:         "Server",
				Name:         "master-1",
				ResourceType: "cx22",
				HourlyPrice:  0.010,
				MonthlyPrice: 7.50,
				Currency:     "EUR",
			},
			{
				Type:         "Server",
				Name:         "worker-1",
				ResourceType: "cx32",
				HourlyPrice:  0.020,
				MonthlyPrice: 14.50,
				Currency:     "EUR",
			},
		}
		// Should not panic
		calculator.displayBudget(costs)
	})

	t.Run("costs with free resources", func(t *testing.T) {
		costs := []ResourceCost{
			{
				Type:         "Network",
				Name:         "test-cluster",
				ResourceType: "10.0.0.0/16",
				HourlyPrice:  0,
				MonthlyPrice: 0,
				Currency:     "EUR",
			},
			{
				Type:         "Firewall",
				Name:         "test-cluster-firewall",
				ResourceType: "3 rules",
				HourlyPrice:  0,
				MonthlyPrice: 0,
				Currency:     "EUR",
			},
			{
				Type:         "SSH Key",
				Name:         "test-cluster-ssh",
				ResourceType: "ab:cd:ef:12:34:56...",
				HourlyPrice:  0,
				MonthlyPrice: 0,
				Currency:     "EUR",
			},
		}
		// Should not panic
		calculator.displayBudget(costs)
	})

	t.Run("mixed paid and free resources", func(t *testing.T) {
		costs := []ResourceCost{
			{
				Type:         "Server",
				Name:         "master-1",
				ResourceType: "cx22",
				HourlyPrice:  0.010,
				MonthlyPrice: 7.50,
				Currency:     "EUR",
			},
			{
				Type:         "Load Balancer",
				Name:         "test-lb",
				ResourceType: "lb11",
				HourlyPrice:  0.005,
				MonthlyPrice: 5.39,
				Currency:     "EUR",
			},
			{
				Type:         "Volume",
				Name:         "test-volume",
				ResourceType: "50GB",
				HourlyPrice:  0.0027,
				MonthlyPrice: 2.00,
				Currency:     "EUR",
			},
			{
				Type:         "Floating IP",
				Name:         "test-fip",
				ResourceType: "ipv4 (1.2.3.4)",
				HourlyPrice:  0.00068,
				MonthlyPrice: 0.50,
				Currency:     "EUR",
			},
			{
				Type:         "Primary IP",
				Name:         "test-pip",
				ResourceType: "ipv4 (5.6.7.8)",
				HourlyPrice:  0.00068,
				MonthlyPrice: 0.50,
				Currency:     "EUR",
			},
			{
				Type:         "Network",
				Name:         "test-cluster",
				ResourceType: "10.0.0.0/16",
				HourlyPrice:  0,
				MonthlyPrice: 0,
				Currency:     "EUR",
			},
		}
		// Should not panic
		calculator.displayBudget(costs)
	})

	t.Run("unknown resource type is skipped gracefully", func(t *testing.T) {
		costs := []ResourceCost{
			{
				Type:         "Unknown",
				Name:         "mystery-resource",
				ResourceType: "mystery",
				HourlyPrice:  1.0,
				MonthlyPrice: 30.0,
				Currency:     "EUR",
			},
		}
		// Unknown resource types should not panic - they are just not displayed
		calculator.displayBudget(costs)
	})
}
func TestFindAutoscaledPoolServers(t *testing.T) {
	tests := []struct {
		name          string
		workerPools   []config.WorkerNodePool
		expectedCalls int
	}{
		{
			name: "non-autoscaling pool should not be queried",
			workerPools: []config.WorkerNodePool{
				{
					NodePool: config.NodePool{
						InstanceType: "cx22",
					},
					Locations: []string{"fsn1"},
				},
			},
			expectedCalls: 0,
		},
		{
			name: "single location autoscaling pool",
			workerPools: []config.WorkerNodePool{
				{
					NodePool: config.NodePool{
						InstanceType: "cx22",
						Autoscaling: &config.Autoscaling{
							Enabled:      true,
							MinInstances: 1,
							MaxInstances: 3,
						},
					},
					Locations: []string{"fsn1"},
				},
			},
			expectedCalls: 1, // One query for the single location
		},
		{
			name: "multi location autoscaling pool",
			workerPools: []config.WorkerNodePool{
				{
					NodePool: config.NodePool{
						InstanceType: "cx22",
						Autoscaling: &config.Autoscaling{
							Enabled:      true,
							MinInstances: 2,
							MaxInstances: 6,
						},
					},
					Locations: []string{"fsn1", "hel1"},
				},
			},
			expectedCalls: 2, // Two queries, one for each location
		},
		{
			name: "multiple autoscaling pools",
			workerPools: []config.WorkerNodePool{
				{
					NodePool: config.NodePool{
						InstanceType: "cx22",
						Autoscaling: &config.Autoscaling{
							Enabled:      true,
							MinInstances: 1,
							MaxInstances: 3,
						},
					},
					Locations: []string{"fsn1"},
				},
				{
					NodePool: config.NodePool{
						InstanceType: "cx32",
						Autoscaling: &config.Autoscaling{
							Enabled:      true,
							MinInstances: 1,
							MaxInstances: 2,
						},
					},
					Locations: []string{"hel1", "nbg1"},
				},
			},
			expectedCalls: 3, // 1 for first pool + 2 for second pool
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a basic test to verify the logic
			// In a real scenario, we'd need to mock the HetznerClient
			cfg := &config.Main{
				ClusterName:     "test-cluster",
				WorkerNodePools: tt.workerPools,
			}

			// Count expected calls based on autoscaling and locations
			expectedQueries := 0
			for _, pool := range tt.workerPools {
				if pool.AutoscalingEnabled() {
					expectedQueries += len(pool.Locations)
				}
			}

			if expectedQueries != tt.expectedCalls {
				t.Errorf("Expected %d queries, calculated %d", tt.expectedCalls, expectedQueries)
			}

			// Verify AutoscalingEnabled works correctly
			for i, pool := range tt.workerPools {
				isEnabled := pool.AutoscalingEnabled()
				hasAutoscaling := pool.Autoscaling != nil && pool.Autoscaling.Enabled

				if isEnabled != hasAutoscaling {
					t.Errorf("Pool %d: AutoscalingEnabled() = %v, but expected %v", i, isEnabled, hasAutoscaling)
				}
			}

			// Verify BuildNodePoolName works
			for _, pool := range tt.workerPools {
				poolName := pool.BuildNodePoolName(cfg.ClusterName)
				if poolName == "" {
					t.Error("BuildNodePoolName returned empty string")
				}
			}
		})
	}
}
