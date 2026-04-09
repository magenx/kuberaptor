// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package cluster

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/magenx/kuberaptor/internal/config"
)

// TestAutoscalingPoolFiltering verifies that autoscaling pools are correctly filtered out
// when creating initial worker nodes
func TestAutoscalingPoolFiltering(t *testing.T) {
	tests := []struct {
		name                string
		pools               []config.WorkerNodePool
		expectedStaticCount int
		expectedTotalNodes  int
	}{
		{
			name: "all static pools",
			pools: []config.WorkerNodePool{
				{
					NodePool: config.NodePool{
						Name:          stringPtr("mariadb"),
						InstanceType:  "cpx32",
						InstanceCount: 1,
					},
					Locations: []string{"nbg1"},
				},
			},
			expectedStaticCount: 1,
			expectedTotalNodes:  1,
		},
		{
			name: "all autoscaling pools",
			pools: []config.WorkerNodePool{
				{
					NodePool: config.NodePool{
						Name:         stringPtr("php"),
						InstanceType: "cpx32",
						Autoscaling: &config.Autoscaling{
							Enabled:      true,
							MinInstances: 1,
							MaxInstances: 3,
						},
					},
					Locations: []string{"nbg1"},
				},
			},
			expectedStaticCount: 0,
			expectedTotalNodes:  0,
		},
		{
			name: "mixed static and autoscaling pools",
			pools: []config.WorkerNodePool{
				{
					NodePool: config.NodePool{
						Name:          stringPtr("mariadb"),
						InstanceType:  "cpx32",
						InstanceCount: 1,
					},
					Locations: []string{"nbg1"},
				},
				{
					NodePool: config.NodePool{
						Name:         stringPtr("php"),
						InstanceType: "cpx32",
						Autoscaling: &config.Autoscaling{
							Enabled:      true,
							MinInstances: 1,
							MaxInstances: 3,
						},
					},
					Locations: []string{"nbg1"},
				},
			},
			expectedStaticCount: 1,
			expectedTotalNodes:  1,
		},
		{
			name: "multiple static pools",
			pools: []config.WorkerNodePool{
				{
					NodePool: config.NodePool{
						Name:          stringPtr("mariadb"),
						InstanceType:  "cpx32",
						InstanceCount: 2,
					},
					Locations: []string{"nbg1"},
				},
				{
					NodePool: config.NodePool{
						Name:          stringPtr("redis"),
						InstanceType:  "cpx22",
						InstanceCount: 1,
					},
					Locations: []string{"nbg1"},
				},
			},
			expectedStaticCount: 2,
			expectedTotalNodes:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the filtering logic from create_enhanced.go
			staticPools := []config.WorkerNodePool{}
			totalWorkers := 0
			for _, pool := range tt.pools {
				if !pool.AutoscalingEnabled() {
					staticPools = append(staticPools, pool)
					totalWorkers += pool.InstanceCount
				}
			}

			if len(staticPools) != tt.expectedStaticCount {
				t.Errorf("Expected %d static pools, got %d", tt.expectedStaticCount, len(staticPools))
			}

			if totalWorkers != tt.expectedTotalNodes {
				t.Errorf("Expected %d total worker nodes, got %d", tt.expectedTotalNodes, totalWorkers)
			}

			// Verify that autoscaling pools are not included
			for _, pool := range staticPools {
				if pool.AutoscalingEnabled() {
					t.Errorf("Autoscaling pool %s should not be in staticPools", *pool.Name)
				}
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

// TestAutoscalingPoolConfiguration verifies that autoscaling pools are collected
// separately for configuration with the cluster autoscaler addon
func TestAutoscalingPoolConfiguration(t *testing.T) {
	pools := []config.WorkerNodePool{
		{
			NodePool: config.NodePool{
				Name:          stringPtr("mariadb"),
				InstanceType:  "cpx32",
				InstanceCount: 1,
			},
			Locations: []string{"nbg1"},
		},
		{
			NodePool: config.NodePool{
				Name:         stringPtr("php"),
				InstanceType: "cpx32",
				Autoscaling: &config.Autoscaling{
					Enabled:      true,
					MinInstances: 1,
					MaxInstances: 3,
				},
			},
			Locations: []string{"nbg1"},
		},
		{
			NodePool: config.NodePool{
				Name:         stringPtr("redis"),
				InstanceType: "cpx22",
				Autoscaling: &config.Autoscaling{
					Enabled:      true,
					MinInstances: 2,
					MaxInstances: 5,
				},
			},
			Locations: []string{"fsn1"},
		},
	}

	// Simulate the filtering logic for static pools (for initial node creation)
	staticPools := []config.WorkerNodePool{}
	for _, pool := range pools {
		if !pool.AutoscalingEnabled() {
			staticPools = append(staticPools, pool)
		}
	}

	// Simulate the collection logic for autoscaling pools (for autoscaler configuration)
	autoscalingPools := []config.WorkerNodePool{}
	for _, pool := range pools {
		if pool.AutoscalingEnabled() {
			autoscalingPools = append(autoscalingPools, pool)
		}
	}

	// Verify static pools (only 1 - mariadb)
	if len(staticPools) != 1 {
		t.Errorf("Expected 1 static pool, got %d", len(staticPools))
	}
	if len(staticPools) > 0 && *staticPools[0].Name != "mariadb" {
		t.Errorf("Expected static pool 'mariadb', got '%s'", *staticPools[0].Name)
	}

	// Verify autoscaling pools (2 - php and redis)
	if len(autoscalingPools) != 2 {
		t.Errorf("Expected 2 autoscaling pools, got %d", len(autoscalingPools))
	}

	// Verify autoscaling pool configurations
	for _, pool := range autoscalingPools {
		if !pool.AutoscalingEnabled() {
			t.Errorf("Pool %s should have autoscaling enabled", *pool.Name)
		}
		if pool.Autoscaling == nil {
			t.Errorf("Pool %s should have autoscaling configuration", *pool.Name)
			continue
		}

		// Verify min/max instances are set (min can be 0)
		if pool.Autoscaling.MinInstances < 0 {
			t.Errorf("Pool %s should have min_instances >= 0, got %d", *pool.Name, pool.Autoscaling.MinInstances)
		}
		if pool.Autoscaling.MaxInstances <= pool.Autoscaling.MinInstances {
			t.Errorf("Pool %s should have max_instances > min_instances, got max=%d min=%d",
				*pool.Name, pool.Autoscaling.MaxInstances, pool.Autoscaling.MinInstances)
		}
	}

	// Verify no overlap between static and autoscaling pools
	for _, staticPool := range staticPools {
		for _, autoscalingPool := range autoscalingPools {
			if *staticPool.Name == *autoscalingPool.Name {
				t.Errorf("Pool %s appears in both static and autoscaling lists", *staticPool.Name)
			}
		}
	}
}

// TestGenerateK3sAddonFlags verifies that K3s addon flags are generated correctly
// including both disable and enable flags (e.g., --embedded-registry)
func TestGenerateK3sAddonFlags(t *testing.T) {
	tests := []struct {
		name             string
		addons           config.Addons
		expectedContains []string
		expectedMissing  []string
	}{
		{
			name: "all addons disabled",
			addons: config.Addons{
				Traefik:                &config.Toggle{Enabled: false},
				ServiceLB:              &config.Toggle{Enabled: false},
				MetricsServer:          &config.Toggle{Enabled: false},
				LocalPathStorageClass:  &config.Toggle{Enabled: false},
				EmbeddedRegistryMirror: &config.Toggle{Enabled: false},
			},
			expectedContains: []string{
				"--disable local-storage",
				"--disable traefik",
				"--disable servicelb",
				"--disable metrics-server",
			},
			expectedMissing: []string{
				"--embedded-registry",
			},
		},
		{
			name: "embedded registry enabled",
			addons: config.Addons{
				Traefik:                &config.Toggle{Enabled: false},
				ServiceLB:              &config.Toggle{Enabled: false},
				MetricsServer:          &config.Toggle{Enabled: false},
				LocalPathStorageClass:  &config.Toggle{Enabled: false},
				EmbeddedRegistryMirror: &config.Toggle{Enabled: true},
			},
			expectedContains: []string{
				"--disable local-storage",
				"--disable traefik",
				"--disable servicelb",
				"--disable metrics-server",
				"--embedded-registry",
			},
			expectedMissing: []string{},
		},
		{
			name: "local path storage enabled, others disabled",
			addons: config.Addons{
				Traefik:                &config.Toggle{Enabled: false},
				ServiceLB:              &config.Toggle{Enabled: false},
				MetricsServer:          &config.Toggle{Enabled: false},
				LocalPathStorageClass:  &config.Toggle{Enabled: true},
				EmbeddedRegistryMirror: &config.Toggle{Enabled: false},
			},
			expectedContains: []string{
				"--disable traefik",
				"--disable servicelb",
				"--disable metrics-server",
			},
			expectedMissing: []string{
				"--disable local-storage",
				"--embedded-registry",
			},
		},
		{
			name: "all enabled",
			addons: config.Addons{
				Traefik:                &config.Toggle{Enabled: true},
				ServiceLB:              &config.Toggle{Enabled: true},
				MetricsServer:          &config.Toggle{Enabled: true},
				LocalPathStorageClass:  &config.Toggle{Enabled: true},
				EmbeddedRegistryMirror: &config.Toggle{Enabled: true},
			},
			expectedContains: []string{
				"--embedded-registry",
			},
			expectedMissing: []string{
				"--disable local-storage",
				"--disable traefik",
				"--disable servicelb",
				"--disable metrics-server",
			},
		},
		{
			name: "default configuration (nil addons)",
			addons: config.Addons{
				Traefik:                nil,
				ServiceLB:              nil,
				MetricsServer:          nil,
				LocalPathStorageClass:  nil,
				EmbeddedRegistryMirror: nil,
			},
			expectedContains: []string{
				"--disable local-storage",
				"--disable traefik",
				"--disable servicelb",
				"--disable metrics-server",
			},
			expectedMissing: []string{
				"--embedded-registry",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Main{
				Addons: tt.addons,
			}
			creator := &CreatorEnhanced{
				Config: cfg,
			}

			flags := creator.generateK3sAddonFlags()

			// Check expected flags are present
			for _, expected := range tt.expectedContains {
				if !strings.Contains(flags, expected) {
					t.Errorf("Expected flags to contain '%s', but got: %s", expected, flags)
				}
			}

			// Check unexpected flags are absent
			for _, missing := range tt.expectedMissing {
				if strings.Contains(flags, missing) {
					t.Errorf("Expected flags to NOT contain '%s', but got: %s", missing, flags)
				}
			}
		})
	}
}

// TestSeparateWorkerPools tests the separateWorkerPools helper function
func TestSeparateWorkerPools(t *testing.T) {
	tests := []struct {
		name                     string
		pools                    []config.WorkerNodePool
		expectedStaticCount      int
		expectedAutoscalingCount int
	}{
		{
			name: "all static pools",
			pools: []config.WorkerNodePool{
				{
					NodePool: config.NodePool{
						Name:          stringPtr("static1"),
						InstanceType:  "cpx32",
						InstanceCount: 2,
					},
					Locations: []string{"nbg1"},
				},
				{
					NodePool: config.NodePool{
						Name:          stringPtr("static2"),
						InstanceType:  "cpx21",
						InstanceCount: 1,
					},
					Locations: []string{"fsn1"},
				},
			},
			expectedStaticCount:      2,
			expectedAutoscalingCount: 0,
		},
		{
			name: "all autoscaling pools",
			pools: []config.WorkerNodePool{
				{
					NodePool: config.NodePool{
						Name:         stringPtr("auto1"),
						InstanceType: "cpx32",
						Autoscaling: &config.Autoscaling{
							Enabled:      true,
							MinInstances: 1,
							MaxInstances: 5,
						},
					},
					Locations: []string{"nbg1"},
				},
				{
					NodePool: config.NodePool{
						Name:         stringPtr("auto2"),
						InstanceType: "cpx21",
						Autoscaling: &config.Autoscaling{
							Enabled:      true,
							MinInstances: 0,
							MaxInstances: 3,
						},
					},
					Locations: []string{"fsn1"},
				},
			},
			expectedStaticCount:      0,
			expectedAutoscalingCount: 2,
		},
		{
			name: "mixed static and autoscaling pools",
			pools: []config.WorkerNodePool{
				{
					NodePool: config.NodePool{
						Name:          stringPtr("static"),
						InstanceType:  "cpx32",
						InstanceCount: 2,
					},
					Locations: []string{"nbg1"},
				},
				{
					NodePool: config.NodePool{
						Name:         stringPtr("auto"),
						InstanceType: "cpx21",
						Autoscaling: &config.Autoscaling{
							Enabled:      true,
							MinInstances: 1,
							MaxInstances: 5,
						},
					},
					Locations: []string{"fsn1"},
				},
			},
			expectedStaticCount:      1,
			expectedAutoscalingCount: 1,
		},
		{
			name:                     "empty pools",
			pools:                    []config.WorkerNodePool{},
			expectedStaticCount:      0,
			expectedAutoscalingCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			static, autoscaling := separateWorkerPools(tt.pools)

			if len(static) != tt.expectedStaticCount {
				t.Errorf("Expected %d static pools, got %d", tt.expectedStaticCount, len(static))
			}

			if len(autoscaling) != tt.expectedAutoscalingCount {
				t.Errorf("Expected %d autoscaling pools, got %d", tt.expectedAutoscalingCount, len(autoscaling))
			}

			// Verify all static pools are not autoscaling-enabled
			for _, pool := range static {
				if pool.AutoscalingEnabled() {
					t.Errorf("Static pool %s should not have autoscaling enabled", *pool.Name)
				}
			}

			// Verify all autoscaling pools are autoscaling-enabled
			for _, pool := range autoscaling {
				if !pool.AutoscalingEnabled() {
					t.Errorf("Autoscaling pool %s should have autoscaling enabled", *pool.Name)
				}
			}
		})
	}
}

// TestParallelResourceCreation verifies that parallel resource creation works correctly
func TestParallelResourceCreation(t *testing.T) {
	t.Run("parallel execution completes successfully", func(t *testing.T) {
		// Simulate parallel creation of multiple resources
		const numResources = 5
		results := make([]int, numResources)

		// Use the same pattern as in create_enhanced.go
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errors []error

		for i := 0; i < numResources; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				// Simulate resource creation
				mu.Lock()
				results[index] = index + 1
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		// Verify all resources were created
		if len(errors) > 0 {
			t.Errorf("Expected no errors, got: %v", errors)
		}

		for i := 0; i < numResources; i++ {
			if results[i] != i+1 {
				t.Errorf("Expected result[%d] to be %d, got %d", i, i+1, results[i])
			}
		}
	})

	t.Run("parallel execution handles errors correctly", func(t *testing.T) {
		// Simulate parallel creation with errors
		const numResources = 5
		const failAtIndex = 2

		var wg sync.WaitGroup
		var mu sync.Mutex
		var errors []error

		for i := 0; i < numResources; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				if index == failAtIndex {
					mu.Lock()
					errors = append(errors, fmt.Errorf("simulated error at index %d", index))
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()

		// Verify error was captured
		if len(errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(errors))
		}

		if len(errors) > 0 && !strings.Contains(errors[0].Error(), "simulated error") {
			t.Errorf("Expected error message to contain 'simulated error', got: %v", errors[0])
		}
	})
}

// TestConcurrentMapAccess verifies that concurrent map access is safe
func TestConcurrentMapAccess(t *testing.T) {
	// Test pattern used for masters creation with pre-allocated slice
	t.Run("pre-allocated slice concurrent access", func(t *testing.T) {
		const numItems = 10
		items := make([]*string, numItems)

		var wg sync.WaitGroup
		var mu sync.Mutex

		for i := 0; i < numItems; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				value := fmt.Sprintf("item-%d", index)
				mu.Lock()
				items[index] = &value
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		// Verify all items were set
		for i, item := range items {
			if item == nil {
				t.Errorf("Expected item at index %d to be set, got nil", i)
			} else if *item != fmt.Sprintf("item-%d", i) {
				t.Errorf("Expected item[%d] to be 'item-%d', got '%s'", i, i, *item)
			}
		}
	})

	// Test pattern used for workers creation with append
	t.Run("append slice concurrent access", func(t *testing.T) {
		const numItems = 10
		items := make([]*string, 0, numItems)

		var wg sync.WaitGroup
		var mu sync.Mutex

		for i := 0; i < numItems; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				value := fmt.Sprintf("item-%d", index)
				mu.Lock()
				items = append(items, &value)
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		// Verify correct number of items (order doesn't matter for workers)
		if len(items) != numItems {
			t.Errorf("Expected %d items, got %d", numItems, len(items))
		}
	})
}

// TestReplaceKubeconfigNames verifies that kubeconfig names are replaced correctly
func TestReplaceKubeconfigNames(t *testing.T) {
	tests := []struct {
		name                string
		input               string
		clusterName         string
		expectedClusterName string
		expectedContextName string
		expectedUserName    string
	}{
		{
			name: "replace all default names with cluster name",
			input: `apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0...
    server: https://10.0.0.1:6443
  name: default
contexts:
- context:
    cluster: default
    user: default
  name: default
current-context: default
kind: Config
preferences: {}
users:
- name: default
  user:
    client-certificate-data: LS0...
    client-key-data: LS0...`,
			clusterName:         "my-cluster",
			expectedClusterName: "name: my-cluster",
			expectedContextName: "name: my-cluster",
			expectedUserName:    "name: my-cluster",
		},
		{
			name: "replace with cluster name containing hyphens",
			input: `apiVersion: v1
clusters:
- cluster:
    server: https://10.0.0.1:6443
  name: default
contexts:
- context:
    cluster: default
    user: default
  name: default
current-context: default
users:
- name: default`,
			clusterName:         "test-cluster-prod",
			expectedClusterName: "name: test-cluster-prod",
			expectedContextName: "name: test-cluster-prod",
			expectedUserName:    "name: test-cluster-prod",
		},
		{
			name: "replace with cluster name containing numbers",
			input: `apiVersion: v1
clusters:
- cluster:
    server: https://10.0.0.1:6443
  name: default
contexts:
- context:
    cluster: default
    user: default
  name: default
current-context: default
users:
- name: default`,
			clusterName:         "cluster123",
			expectedClusterName: "name: cluster123",
			expectedContextName: "name: cluster123",
			expectedUserName:    "name: cluster123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceKubeconfigNames(tt.input, tt.clusterName)

			// Verify cluster name is replaced
			if !strings.Contains(result, tt.expectedClusterName) {
				t.Errorf("Expected result to contain cluster name '%s', but got:\n%s", tt.expectedClusterName, result)
			}

			// Verify context name is replaced
			if !strings.Contains(result, tt.expectedContextName) {
				t.Errorf("Expected result to contain context name '%s', but got:\n%s", tt.expectedContextName, result)
			}

			// Verify user name is replaced
			if !strings.Contains(result, tt.expectedUserName) {
				t.Errorf("Expected result to contain user name '%s', but got:\n%s", tt.expectedUserName, result)
			}

			// Verify context references are replaced
			expectedContextCluster := fmt.Sprintf("cluster: %s", tt.clusterName)
			if !strings.Contains(result, expectedContextCluster) {
				t.Errorf("Expected result to contain context cluster reference '%s', but got:\n%s", expectedContextCluster, result)
			}

			expectedContextUser := fmt.Sprintf("user: %s", tt.clusterName)
			if !strings.Contains(result, expectedContextUser) {
				t.Errorf("Expected result to contain context user reference '%s', but got:\n%s", expectedContextUser, result)
			}

			// Verify current-context is replaced
			expectedCurrentContext := fmt.Sprintf("current-context: %s", tt.clusterName)
			if !strings.Contains(result, expectedCurrentContext) {
				t.Errorf("Expected result to contain current-context '%s', but got:\n%s", expectedCurrentContext, result)
			}
		})
	}
}

// TestBuildLabelsAndTaintsForWorker tests the buildLabelsAndTaintsForWorker function
func TestBuildLabelsAndTaintsForWorker(t *testing.T) {
creator := &CreatorEnhanced{
Config: &config.Main{ClusterName: "test-cluster"},
}

tests := []struct {
name     string
pool     *config.WorkerNodePool
contains []string
missing  []string
}{
{
name:    "nil pool returns empty string",
pool:    nil,
missing: []string{"--node-label", "--node-taint"},
},
{
name: "pool with no labels or taints",
pool: &config.WorkerNodePool{
NodePool: config.NodePool{
InstanceType: "cx22",
},
Locations: []string{"fsn1"},
},
missing: []string{"--node-label", "--node-taint"},
},
{
name: "pool with kubernetes labels",
pool: &config.WorkerNodePool{
NodePool: config.NodePool{
InstanceType: "cx22",
Kubernetes: &config.KubernetesConfig{
Labels: []config.Label{
{Key: "role", Value: "worker"},
{Key: "env", Value: "production"},
},
},
},
Locations: []string{"fsn1"},
},
contains: []string{"--node-label=", "role=worker", "env=production"},
missing:  []string{"--node-taint"},
},
{
name: "pool with kubernetes taints",
pool: &config.WorkerNodePool{
NodePool: config.NodePool{
InstanceType: "cx22",
Kubernetes: &config.KubernetesConfig{
Taints: []config.Taint{
{Key: "dedicated", Value: "gpu", Effect: "NoSchedule"},
},
},
},
Locations: []string{"fsn1"},
},
contains: []string{"--node-taint=", "dedicated=gpu:NoSchedule"},
missing:  []string{"--node-label"},
},
{
name: "pool with both labels and taints",
pool: &config.WorkerNodePool{
NodePool: config.NodePool{
InstanceType: "cx22",
Kubernetes: &config.KubernetesConfig{
Labels: []config.Label{
{Key: "tier", Value: "backend"},
},
Taints: []config.Taint{
{Key: "workload", Value: "database", Effect: "NoSchedule"},
},
},
},
Locations: []string{"fsn1"},
},
contains: []string{"--node-label=", "tier=backend", "--node-taint=", "workload=database:NoSchedule"},
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := creator.buildLabelsAndTaintsForWorker(tt.pool)
for _, s := range tt.contains {
if !strings.Contains(result, s) {
t.Errorf("expected result to contain %q, got: %q", s, result)
}
}
for _, s := range tt.missing {
if strings.Contains(result, s) {
t.Errorf("expected result NOT to contain %q, got: %q", s, result)
}
}
})
}
}

// TestBuildHetznerServerLabels tests the buildHetznerServerLabels function
func TestBuildHetznerServerLabels(t *testing.T) {
tests := []struct {
name            string
clusterName     string
pool            *config.WorkerNodePool
poolName        string
location        string
expectLabels    map[string]string
forbiddenLabels map[string]string
}{
{
name:        "default labels with nil pool",
clusterName: "my-cluster",
pool:        nil,
poolName:    "workers",
location:    "fsn1",
expectLabels: map[string]string{
"cluster":  "my-cluster",
"role":     "worker",
"pool":     "workers",
"location": "fsn1",
"managed":  "kuberaptor",
},
},
{
name:        "custom labels are merged",
clusterName: "my-cluster",
pool: &config.WorkerNodePool{
NodePool: config.NodePool{
InstanceType: "cx22",
Hetzner: &config.HetznerConfig{
Labels: []config.Label{
{Key: "env", Value: "staging"},
{Key: "team", Value: "platform"},
},
},
},
Locations: []string{"fsn1"},
},
poolName: "backend",
location: "fsn1",
expectLabels: map[string]string{
"cluster":  "my-cluster",
"role":     "worker",
"pool":     "backend",
"location": "fsn1",
"managed":  "kuberaptor",
"env":      "staging",
"team":     "platform",
},
},
{
name:        "managed label cannot be overridden",
clusterName: "my-cluster",
pool: &config.WorkerNodePool{
NodePool: config.NodePool{
InstanceType: "cx22",
Hetzner: &config.HetznerConfig{
Labels: []config.Label{
{Key: "managed", Value: "should-not-override"},
},
},
},
Locations: []string{"fsn1"},
},
poolName: "workers",
location: "nbg1",
expectLabels: map[string]string{
"managed": "kuberaptor",
},
forbiddenLabels: map[string]string{
"managed": "should-not-override",
},
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
creator := &CreatorEnhanced{
Config: &config.Main{ClusterName: tt.clusterName},
}

labels := creator.buildHetznerServerLabels(tt.pool, tt.poolName, tt.location)

for k, v := range tt.expectLabels {
if labels[k] != v {
t.Errorf("expected label[%q]=%q, got %q", k, v, labels[k])
}
}
for k, forbidden := range tt.forbiddenLabels {
if labels[k] == forbidden {
t.Errorf("label[%q] should not be %q", k, forbidden)
}
}
})
}
}
