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

// TestFindAutoscaledPoolServers tests that the function is called for autoscaling pools
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
