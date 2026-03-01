package addons

import (
	"fmt"
	"testing"

	"github.com/magenx/kuberaptor/internal/config"
)

// TestAutoscalerNodeGroupNaming tests the node group naming logic for multi-location pools
func TestAutoscalerNodeGroupNaming(t *testing.T) {
	tests := []struct {
		name              string
		poolName          string
		clusterName       string
		locations         []string
		expectSuffix      bool
		expectedNodeGroup string
	}{
		{
			name:              "single location - no suffix",
			poolName:          "workers",
			clusterName:       "test-cluster",
			locations:         []string{"fsn1"},
			expectSuffix:      false,
			expectedNodeGroup: "test-cluster-workers",
		},
		{
			name:              "multi-location - with suffix",
			poolName:          "workers",
			clusterName:       "test-cluster",
			locations:         []string{"fsn1", "hel1"},
			expectSuffix:      true,
			expectedNodeGroup: "test-cluster-workers-hel1", // for last location (index 1)
		},
		{
			name:              "three locations - with suffix",
			poolName:          "gpu",
			clusterName:       "prod",
			locations:         []string{"fsn1", "hel1", "nbg1"},
			expectSuffix:      true,
			expectedNodeGroup: "prod-gpu-nbg1", // for third location
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := config.WorkerNodePool{
				NodePool: config.NodePool{
					Name:                       &tt.poolName,
					IncludeClusterNameAsPrefix: true,
				},
				Locations: tt.locations,
			}

			basePoolName := pool.BuildNodePoolName(tt.clusterName)

			// Simulate the naming logic from patchAutoscalerContainer
			for i, location := range tt.locations {
				poolName := basePoolName
				if len(tt.locations) > 1 {
					poolName = fmt.Sprintf("%s-%s", basePoolName, location)
				}

				// Test only the expected node group
				if (i == 0 && !tt.expectSuffix) ||
					(i == len(tt.locations)-1 && tt.expectSuffix) {
					if poolName != tt.expectedNodeGroup {
						t.Errorf("Expected node group name '%s', got '%s'", tt.expectedNodeGroup, poolName)
					}
				}
			}
		})
	}
}

// TestAutoscalerInstanceDistribution tests the distribution of min/max instances across locations
func TestAutoscalerInstanceDistribution(t *testing.T) {
	tests := []struct {
		name             string
		minInstances     int
		maxInstances     int
		numLocations     int
		expectedFirstMin int
		expectedFirstMax int
		expectedOtherMin int
		expectedOtherMax int
	}{
		{
			name:             "evenly divisible",
			minInstances:     3,
			maxInstances:     9,
			numLocations:     3,
			expectedFirstMin: 1, // 3/3 + 0 remainder
			expectedFirstMax: 3, // 9/3 + 0 remainder
			expectedOtherMin: 1, // 3/3
			expectedOtherMax: 3, // 9/3
		},
		{
			name:             "with remainder",
			minInstances:     5,
			maxInstances:     10,
			numLocations:     3,
			expectedFirstMin: 3, // 5/3=1 + 2 remainder
			expectedFirstMax: 4, // 10/3=3 + 1 remainder
			expectedOtherMin: 1, // 5/3=1
			expectedOtherMax: 3, // 10/3=3
		},
		{
			name:             "single location",
			minInstances:     5,
			maxInstances:     10,
			numLocations:     1,
			expectedFirstMin: 5,  // 5/1 + 0 remainder
			expectedFirstMax: 10, // 10/1 + 0 remainder
			expectedOtherMin: 0,  // not applicable
			expectedOtherMax: 0,  // not applicable
		},
		{
			name:             "zero min instances",
			minInstances:     0,
			maxInstances:     6,
			numLocations:     2,
			expectedFirstMin: 0, // 0/2 + 0 remainder
			expectedFirstMax: 3, // 6/2 + 0 remainder
			expectedOtherMin: 0, // 0/2
			expectedOtherMax: 3, // 6/2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the distribution logic from patchAutoscalerContainer
			for i := 0; i < tt.numLocations; i++ {
				minInstances := tt.minInstances / tt.numLocations
				maxInstances := tt.maxInstances / tt.numLocations

				// First location gets the remainder
				if i == 0 {
					minInstances += tt.minInstances % tt.numLocations
					maxInstances += tt.maxInstances % tt.numLocations
				}

				// Verify first location
				if i == 0 {
					if minInstances != tt.expectedFirstMin {
						t.Errorf("First location min: expected %d, got %d", tt.expectedFirstMin, minInstances)
					}
					if maxInstances != tt.expectedFirstMax {
						t.Errorf("First location max: expected %d, got %d", tt.expectedFirstMax, maxInstances)
					}
				} else {
					// Verify other locations
					if minInstances != tt.expectedOtherMin {
						t.Errorf("Other location min: expected %d, got %d", tt.expectedOtherMin, minInstances)
					}
					if maxInstances != tt.expectedOtherMax {
						t.Errorf("Other location max: expected %d, got %d", tt.expectedOtherMax, maxInstances)
					}
				}
			}

			// Verify total capacity is preserved
			totalMin := tt.expectedFirstMin + (tt.expectedOtherMin * (tt.numLocations - 1))
			totalMax := tt.expectedFirstMax + (tt.expectedOtherMax * (tt.numLocations - 1))

			if totalMin != tt.minInstances {
				t.Errorf("Total min instances mismatch: expected %d, got %d", tt.minInstances, totalMin)
			}
			if totalMax != tt.maxInstances {
				t.Errorf("Total max instances mismatch: expected %d, got %d", tt.maxInstances, totalMax)
			}
		})
	}
}
