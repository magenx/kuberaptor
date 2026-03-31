// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package config

import (
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGenerateSkeleton(t *testing.T) {
	// Generate the skeleton
	data, err := GenerateSkeleton()
	if err != nil {
		t.Fatalf("GenerateSkeleton() error = %v", err)
	}

	// Verify it's valid YAML
	var result map[string]interface{}
	if err := yaml.Unmarshal(data, &result); err != nil {
		t.Fatalf("Generated YAML is not valid: %v", err)
	}

	// Check that main fields exist
	expectedFields := []string{
		"hetzner_token",
		"cluster_name",
		"kubeconfig_path",
		"k3s_version",
		"masters_pool",
		"networking",
		"datastore",
		"addons",
	}

	for _, field := range expectedFields {
		if _, exists := result[field]; !exists {
			t.Errorf("Expected field %q not found in generated skeleton", field)
		}
	}

	// Verify the output as string contains expected structure
	yamlStr := string(data)

	// Check for nested structures
	if !strings.Contains(yamlStr, "networking:") {
		t.Error("Expected 'networking:' section in generated YAML")
	}
	if !strings.Contains(yamlStr, "datastore:") {
		t.Error("Expected 'datastore:' section in generated YAML")
	}
	if !strings.Contains(yamlStr, "ssh:") {
		t.Error("Expected 'ssh:' section under networking in generated YAML")
	}
}

func TestGenerateStructSkeleton(t *testing.T) {
	// Test with SSH struct
	skeleton := generateStructSkeleton(reflect.TypeOf(SSH{}))

	sshMap, ok := skeleton.(map[string]interface{})
	if !ok {
		t.Fatal("Expected skeleton to be a map")
	}

	expectedFields := []string{"port", "use_agent", "private_key_path", "public_key_path"}
	for _, field := range expectedFields {
		if _, exists := sshMap[field]; !exists {
			t.Errorf("Expected field %q not found in SSH skeleton", field)
		}
	}
}

func TestGenerateFieldValue(t *testing.T) {
	tests := []struct {
		name      string
		fieldType reflect.Type
		wantType  string
	}{
		{
			name:      "string field",
			fieldType: reflect.TypeOf(""),
			wantType:  "string",
		},
		{
			name:      "int field",
			fieldType: reflect.TypeOf(0),
			wantType:  "int",
		},
		{
			name:      "bool field",
			fieldType: reflect.TypeOf(false),
			wantType:  "bool",
		},
		{
			name:      "struct field",
			fieldType: reflect.TypeOf(SSH{}),
			wantType:  "map",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateFieldValue(tt.fieldType)

			var gotType string
			switch result.(type) {
			case string:
				gotType = "string"
			case int, int64:
				gotType = "int"
			case bool:
				gotType = "bool"
			case map[string]interface{}:
				gotType = "map"
			default:
				gotType = "unknown"
			}

			if gotType != tt.wantType {
				t.Errorf("generateFieldValue() type = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"HetznerToken", "hetzner_token"},
		{"ClusterName", "cluster_name"},
		{"K3sVersion", "k3s_version"},
		{"APIServerHostname", "api_server_hostname"},
		{"SSH", "ssh"},
		{"SSHPort", "ssh_port"},
		{"HTTPSEnabled", "https_enabled"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	slice := []string{"inline", "omitempty", "flow"}

	if !contains(slice, "inline") {
		t.Error("Expected contains to find 'inline'")
	}
	if contains(slice, "notfound") {
		t.Error("Expected contains to not find 'notfound'")
	}
}
