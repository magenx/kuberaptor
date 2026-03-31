// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package config

import (
	"strings"
	"testing"
)

// TestAutoscalingPoolNameGeneration tests that autoscaling pool names are generated correctly
func TestAutoscalingPoolNameGeneration(t *testing.T) {
	tests := []struct {
		name                       string
		clusterName                string
		poolName                   string
		includeClusterNameAsPrefix bool
		expected                   string
	}{
		{
			name:                       "with cluster prefix",
			clusterName:                "magenx",
			poolName:                   "php",
			includeClusterNameAsPrefix: true,
			expected:                   "magenx-php",
		},
		{
			name:                       "without cluster prefix",
			clusterName:                "magenx",
			poolName:                   "php",
			includeClusterNameAsPrefix: false,
			expected:                   "php",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := WorkerNodePool{
				NodePool: NodePool{
					Name:                       &tt.poolName,
					InstanceType:               "cpx32",
					IncludeClusterNameAsPrefix: tt.includeClusterNameAsPrefix,
					Autoscaling: &Autoscaling{
						Enabled:      true,
						MinInstances: 1,
						MaxInstances: 3,
					},
				},
				Locations: []string{"nbg1"},
			}

			// Test the BuildNodePoolName method
			result := pool.BuildNodePoolName(tt.clusterName)

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}

			// Call SetDefaults to migrate Location to Locations array
			pool.SetDefaults()

			// Also test the node pool argument format
			// After SetDefaults, Location is migrated to Locations array
			nodePoolArg := "--nodes=" + "1:3:" + strings.ToUpper(pool.InstanceType) + ":" + strings.ToUpper(pool.Locations[0]) + ":" + result
			expectedArg := "--nodes=1:3:CPX32:NBG1:" + tt.expected

			if nodePoolArg != expectedArg {
				t.Errorf("Expected node pool arg %s, got %s", expectedArg, nodePoolArg)
			}
		})
	}
}

// TestAutoscalingPoolDefaults tests that defaults are set correctly
func TestAutoscalingPoolDefaults(t *testing.T) {
	phpName := "php"
	pool := WorkerNodePool{
		NodePool: NodePool{
			Name:         &phpName,
			InstanceType: "cpx32",
			Autoscaling: &Autoscaling{
				Enabled:      true,
				MinInstances: 1,
				MaxInstances: 3,
			},
		},
		Locations: []string{"nbg1"},
	}

	// Before SetDefaults
	if pool.IncludeClusterNameAsPrefix {
		t.Error("IncludeClusterNameAsPrefix should be false before SetDefaults")
	}

	// Call SetDefaults
	pool.SetDefaults()

	// After SetDefaults
	if !pool.IncludeClusterNameAsPrefix {
		t.Error("IncludeClusterNameAsPrefix should be true after SetDefaults")
	}

	// Verify instance_count is not set for autoscaling pools
	if pool.InstanceCount != 0 {
		t.Errorf("Instance count should be 0 for autoscaling pools, got %d", pool.InstanceCount)
	}
}

// TestAutoscalingPoolWithZeroMinInstances tests that autoscaling pools can have min_instances: 0
func TestAutoscalingPoolWithZeroMinInstances(t *testing.T) {
	poolName := "zero-min-pool"
	config := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		WorkerNodePools: []WorkerNodePool{
			{
				NodePool: NodePool{
					Name:         &poolName,
					InstanceType: "cpx32",
					Autoscaling: &Autoscaling{
						Enabled:      true,
						MinInstances: 0, // Should be valid - pool starts with 0 nodes
						MaxInstances: 5,
					},
				},
				Locations: []string{"nbg1"},
			},
		},
	}

	// Validate configuration
	validator := NewValidator(config)
	validator.validateWorkerPools()

	// Should have no errors
	if len(validator.GetErrors()) > 0 {
		t.Errorf("Expected no validation errors for min_instances: 0, got: %v", validator.GetErrors())
	}

	// Verify the pool is recognized as autoscaling-enabled
	pool := config.WorkerNodePools[0]
	if !pool.AutoscalingEnabled() {
		t.Error("Pool should be recognized as autoscaling-enabled")
	}

	// Verify that instance_count is 0 (no initial workers should be created)
	if pool.InstanceCount != 0 {
		t.Errorf("Instance count should be 0 for autoscaling pools, got %d", pool.InstanceCount)
	}
}
