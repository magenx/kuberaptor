// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package config

import (
	"testing"
)

// TestWorkerPoolMultiLocationSupport tests that worker pools support multiple locations
func TestWorkerPoolMultiLocationSupport(t *testing.T) {
	phpName := "multi-region"
	pool := WorkerNodePool{
		NodePool: NodePool{
			Name:         &phpName,
			InstanceType: "cpx32",
		},
		Locations: []string{"fsn1", "hel1", "nbg1"},
	}

	pool.SetDefaults()

	// Verify locations are preserved
	if len(pool.Locations) != 3 {
		t.Errorf("Expected 3 locations, got %d", len(pool.Locations))
	}

	expectedLocations := []string{"fsn1", "hel1", "nbg1"}
	for i, loc := range pool.Locations {
		if loc != expectedLocations[i] {
			t.Errorf("Expected location %s at index %d, got %s", expectedLocations[i], i, loc)
		}
	}
}

// TestWorkerPoolDefaultLocation tests that default location is set when none specified
func TestWorkerPoolDefaultLocation(t *testing.T) {
	phpName := "default-test"
	pool := WorkerNodePool{
		NodePool: NodePool{
			Name:         &phpName,
			InstanceType: "cpx32",
		},
		// No locations specified
	}

	pool.SetDefaults()

	// Verify default location is set
	if len(pool.Locations) != 1 {
		t.Errorf("Expected 1 default location, got %d", len(pool.Locations))
	}

	if pool.Locations[0] != "fsn1" {
		t.Errorf("Expected default location 'fsn1', got '%s'", pool.Locations[0])
	}
}

// TestMasterPoolMultiLocationDefault tests that master pools default to fsn1
func TestMasterPoolMultiLocationDefault(t *testing.T) {
	pool := MasterNodePool{
		NodePool: NodePool{
			InstanceType: "cpx32",
		},
		// No locations specified
	}

	pool.SetDefaults()

	// Verify default location is set
	if len(pool.Locations) != 1 {
		t.Errorf("Expected 1 default location, got %d", len(pool.Locations))
	}

	if pool.Locations[0] != "fsn1" {
		t.Errorf("Expected default location 'fsn1', got '%s'", pool.Locations[0])
	}
}

// TestMasterPoolMultiLocationPreserved tests that master pool locations are preserved
func TestMasterPoolMultiLocationPreserved(t *testing.T) {
	pool := MasterNodePool{
		NodePool: NodePool{
			InstanceType: "cpx32",
		},
		Locations: []string{"fsn1", "hel1", "nbg1"},
	}

	pool.SetDefaults()

	// Verify locations are preserved
	if len(pool.Locations) != 3 {
		t.Errorf("Expected 3 locations, got %d", len(pool.Locations))
	}

	expectedLocations := []string{"fsn1", "hel1", "nbg1"}
	for i, loc := range pool.Locations {
		if loc != expectedLocations[i] {
			t.Errorf("Expected location %s at index %d, got %s", expectedLocations[i], i, loc)
		}
	}
}
