// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"strings"
	"testing"
)

func TestValidateWorkerPools_WithAutoscaling(t *testing.T) {
	// Test case: Worker pool with autoscaling enabled should NOT require instance_count
	phpPoolName := "php"
	config := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		WorkerNodePools: []WorkerNodePool{
			{
				NodePool: NodePool{
					Name:          &phpPoolName,
					InstanceType:  "cpx32",
					InstanceCount: 0, // This should be ignored when autoscaling is enabled
					Autoscaling: &Autoscaling{
						Enabled:      true,
						MinInstances: 1,
						MaxInstances: 3,
					},
				},
				Locations: []string{"nbg1"},
			},
		},
	}

	validator := NewValidator(config)
	validator.validateWorkerPools()

	// Should not have any errors about instance_count
	for _, err := range validator.GetErrors() {
		if strings.Contains(err, "instance_count") {
			t.Errorf("Expected no instance_count validation error when autoscaling is enabled, got: %s", err)
		}
	}
}

func TestValidateWorkerPools_WithoutAutoscaling(t *testing.T) {
	// Test case: Worker pool without autoscaling must have instance_count >= 1
	mariadbPoolName := "mariadb"
	config := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		WorkerNodePools: []WorkerNodePool{
			{
				NodePool: NodePool{
					Name:          &mariadbPoolName,
					InstanceType:  "cpx32",
					InstanceCount: 0, // This should trigger an error
				},
				Locations: []string{"nbg1"},
			},
		},
	}

	validator := NewValidator(config)
	validator.validateWorkerPools()

	// Should have an error about instance_count
	foundError := false
	for _, err := range validator.GetErrors() {
		if strings.Contains(err, "instance_count must be at least 1") {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Error("Expected instance_count validation error when autoscaling is not enabled")
	}
}

func TestValidateWorkerPools_AutoscalingMinInstances(t *testing.T) {
	// Test case: Autoscaling min_instances can be 0
	poolName := "test"
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
						MinInstances: 0, // Valid: can start with 0 nodes and scale up
						MaxInstances: 3,
					},
				},
				Locations: []string{"nbg1"},
			},
		},
	}

	validator := NewValidator(config)
	validator.validateWorkerPools()

	// Should NOT have any errors - min_instances: 0 is valid
	if len(validator.GetErrors()) > 0 {
		t.Errorf("Expected no validation errors for min_instances: 0, got: %v", validator.GetErrors())
	}
}

func TestValidateWorkerPools_AutoscalingMaxGreaterThanMin(t *testing.T) {
	// Test case: Autoscaling max_instances must be greater than min_instances
	poolName := "test"
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
						MinInstances: 3,
						MaxInstances: 3, // Invalid: should be greater than min
					},
				},
				Locations: []string{"nbg1"},
			},
		},
	}

	validator := NewValidator(config)
	validator.validateWorkerPools()

	// Should have an error about max_instances
	foundError := false
	for _, err := range validator.GetErrors() {
		if strings.Contains(err, "max_instances must be greater than min_instances") {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Error("Expected max_instances validation error")
	}
}

func TestValidateWorkerPools_AutoscalingNegativeMinInstances(t *testing.T) {
	// Test case: Autoscaling min_instances cannot be negative
	poolName := "test"
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
						MinInstances: -1, // Invalid: cannot be negative
						MaxInstances: 3,
					},
				},
				Locations: []string{"nbg1"},
			},
		},
	}

	validator := NewValidator(config)
	validator.validateWorkerPools()

	// Should have an error about negative min_instances
	foundError := false
	for _, err := range validator.GetErrors() {
		if strings.Contains(err, "min_instances cannot be negative") {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Error("Expected min_instances negative validation error")
	}
}

func TestValidateWorkerPools_MixedAutoscalingAndStatic(t *testing.T) {
	// Test case: Mix of autoscaling and static pools
	mariadbPoolName := "mariadb"
	phpPoolName := "php"
	config := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		WorkerNodePools: []WorkerNodePool{
			{
				NodePool: NodePool{
					Name:          &mariadbPoolName,
					InstanceType:  "cpx32",
					InstanceCount: 1, // Static pool with explicit count
				},
				Locations: []string{"nbg1"},
			},
			{
				NodePool: NodePool{
					Name:         &phpPoolName,
					InstanceType: "cpx32",
					Autoscaling: &Autoscaling{
						Enabled:      true,
						MinInstances: 1,
						MaxInstances: 3,
					},
				},
				Locations: []string{"nbg1"},
			},
		},
	}

	validator := NewValidator(config)
	validator.validateWorkerPools()

	// Should have no errors
	if len(validator.GetErrors()) > 0 {
		t.Errorf("Expected no errors for valid mixed configuration, got: %v", validator.GetErrors())
	}
}

func TestValidateWorkerPools_ValidAutoscaling(t *testing.T) {
	// Test case: Valid autoscaling configuration
	poolName := "php"
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
						MinInstances: 1,
						MaxInstances: 5,
					},
				},
				Locations: []string{"nbg1"},
			},
		},
	}

	validator := NewValidator(config)
	validator.validateWorkerPools()

	// Should have no errors
	if len(validator.GetErrors()) > 0 {
		t.Errorf("Expected no errors for valid autoscaling configuration, got: %v", validator.GetErrors())
	}
}

func TestValidateSSHKeys_TildeExpansion(t *testing.T) {
	// Test case: SSH key paths with tilde should be properly expanded before validation
	// Create temporary SSH keys for testing
	tmpDir := t.TempDir()
	privateKeyPath := tmpDir + "/id_test"
	publicKeyPath := tmpDir + "/id_test.pub"

	// Create dummy SSH key files
	if err := createDummyFile(privateKeyPath); err != nil {
		t.Fatalf("Failed to create temporary private key: %v", err)
	}
	if err := createDummyFile(publicKeyPath); err != nil {
		t.Fatalf("Failed to create temporary public key: %v", err)
	}

	// Test with absolute paths first - should work
	config := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		Networking: Networking{
			SSH: SSH{
				PrivateKeyPath: privateKeyPath,
				PublicKeyPath:  publicKeyPath,
				Port:           22,
			},
		},
		MastersPool: MasterNodePool{
			NodePool: NodePool{
				InstanceType:  "cpx11",
				InstanceCount: 1,
			},
			Locations: []string{"nbg1"},
		},
	}

	validator := NewValidator(config)
	validator.validateSSHKeys()

	// Should have no errors for valid absolute paths
	sshErrors := filterSSHKeyErrors(validator.GetErrors())
	if len(sshErrors) > 0 {
		t.Errorf("Expected no SSH key validation errors for absolute paths, got: %v", sshErrors)
	}
}

// Helper function to create a dummy file for testing
func createDummyFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString("dummy content")
	return err
}

// Helper function to filter SSH key related errors
func filterSSHKeyErrors(errors []string) []string {
	var sshErrors []string
	for _, err := range errors {
		if strings.Contains(err, "SSH") && (strings.Contains(err, "key") || strings.Contains(err, "path")) {
			sshErrors = append(sshErrors, err)
		}
	}
	return sshErrors
}

func TestValidatePlacementGroup_ValidConfig(t *testing.T) {
	// Test case: Valid placement group configuration should pass
	config := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		MastersPool: MasterNodePool{
			NodePool: NodePool{
				InstanceType:  "cpx22",
				InstanceCount: 3,
				PlacementGroup: &PlacementGroupConfig{
					Name: "master",
					Type: "spread",
					Labels: []Label{
						{Key: "environment", Value: "production"},
					},
				},
			},
			Locations: []string{"nbg1", "fsn1"},
		},
	}

	validator := NewValidator(config)
	validator.validateMasterPool()

	// Should not have any placement group errors
	for _, err := range validator.GetErrors() {
		if strings.Contains(err, "placement_group") {
			t.Errorf("Expected no placement_group validation error for valid config, got: %s", err)
		}
	}
}

func TestValidatePlacementGroup_MissingName(t *testing.T) {
	// Test case: Placement group without name should fail
	config := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		MastersPool: MasterNodePool{
			NodePool: NodePool{
				InstanceType:  "cpx22",
				InstanceCount: 3,
				PlacementGroup: &PlacementGroupConfig{
					Type: "spread",
				},
			},
			Locations: []string{"nbg1"},
		},
	}

	validator := NewValidator(config)
	validator.validateMasterPool()

	// Should have an error about missing name
	foundError := false
	for _, err := range validator.GetErrors() {
		if strings.Contains(err, "placement_group.name is required") {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Error("Expected placement_group.name validation error")
	}
}

func TestValidatePlacementGroup_MissingType(t *testing.T) {
	// Test case: Placement group without type should fail
	config := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		MastersPool: MasterNodePool{
			NodePool: NodePool{
				InstanceType:  "cpx22",
				InstanceCount: 3,
				PlacementGroup: &PlacementGroupConfig{
					Name: "master",
				},
			},
			Locations: []string{"nbg1"},
		},
	}

	validator := NewValidator(config)
	validator.validateMasterPool()

	// Should have an error about missing type
	foundError := false
	for _, err := range validator.GetErrors() {
		if strings.Contains(err, "placement_group.type is required") {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Error("Expected placement_group.type validation error")
	}
}

func TestValidatePlacementGroup_InvalidType(t *testing.T) {
	// Test case: Placement group with invalid type should fail
	config := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		MastersPool: MasterNodePool{
			NodePool: NodePool{
				InstanceType:  "cpx22",
				InstanceCount: 3,
				PlacementGroup: &PlacementGroupConfig{
					Name: "master",
					Type: "invalid",
				},
			},
			Locations: []string{"nbg1"},
		},
	}

	validator := NewValidator(config)
	validator.validateMasterPool()

	// Should have an error about invalid type
	foundError := false
	for _, err := range validator.GetErrors() {
		if strings.Contains(err, "placement_group.type must be 'spread'") {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Error("Expected placement_group.type validation error for invalid type")
	}
}

func TestValidatePlacementGroup_WorkerPool(t *testing.T) {
	// Test case: Worker pool with placement group should be validated
	poolName := "valkey"
	config := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		WorkerNodePools: []WorkerNodePool{
			{
				NodePool: NodePool{
					Name:          &poolName,
					InstanceType:  "cpx22",
					InstanceCount: 3,
					PlacementGroup: &PlacementGroupConfig{
						Name: "valkey",
						Type: "spread",
					},
				},
				Locations: []string{"nbg1"},
			},
		},
	}

	validator := NewValidator(config)
	validator.validateWorkerPools()

	// Should not have any placement group errors
	for _, err := range validator.GetErrors() {
		if strings.Contains(err, "placement_group") {
			t.Errorf("Expected no placement_group validation error for valid worker pool config, got: %s", err)
		}
	}
}

func TestValidatePlacementGroup_EmptyLabelKey(t *testing.T) {
	// Test case: Placement group with empty label key should fail
	config := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		MastersPool: MasterNodePool{
			NodePool: NodePool{
				InstanceType:  "cpx22",
				InstanceCount: 3,
				PlacementGroup: &PlacementGroupConfig{
					Name: "master",
					Type: "spread",
					Labels: []Label{
						{Key: "", Value: "value"},
					},
				},
			},
			Locations: []string{"nbg1"},
		},
	}

	validator := NewValidator(config)
	validator.validateMasterPool()

	// Should have an error about empty label key
	foundError := false
	for _, err := range validator.GetErrors() {
		if strings.Contains(err, "placement_group.labels") && strings.Contains(err, "empty key") {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Error("Expected placement_group.labels validation error for empty key")
	}
}

func TestValidateExternalTools_CiliumWarning(t *testing.T) {
	cfg := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		Networking: Networking{
			CNI: CNI{
				Mode:   "cilium",
				Cilium: &Cilium{Enabled: true},
			},
		},
	}

	validator := NewValidator(cfg)
	validator.validateExternalTools()

	found := false
	for _, w := range validator.GetWarnings() {
		if strings.Contains(w, "cilium-cli") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected a warning about cilium-cli when CNI mode is 'cilium'")
	}
}

func TestValidateExternalTools_NoCiliumWarningForFlannel(t *testing.T) {
	cfg := &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		Networking: Networking{
			CNI: CNI{Mode: "flannel"},
		},
	}

	validator := NewValidator(cfg)
	validator.validateExternalTools()

	for _, w := range validator.GetWarnings() {
		if strings.Contains(w, "cilium") {
			t.Errorf("Expected no Cilium warning for flannel CNI mode, got: %s", w)
		}
	}
}
