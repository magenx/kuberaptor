package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestAPILoadBalancer_UnmarshalYAML_Enabled(t *testing.T) {
	yamlData := `
api_load_balancer:
  enabled: true
`
	type testConfig struct {
		APILoadBalancer APILoadBalancer `yaml:"api_load_balancer"`
	}

	var config testConfig
	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !config.APILoadBalancer.Enabled {
		t.Errorf("Expected APILoadBalancer.Enabled to be true, got false")
	}
}

func TestAPILoadBalancer_UnmarshalYAML_WithHetznerLabels(t *testing.T) {
	yamlData := `
api_load_balancer:
  enabled: true
  hetzner:
    labels:
      - key: cluster_id
        value: "123456"
      - key: environment
        value: production
`
	type testConfig struct {
		APILoadBalancer APILoadBalancer `yaml:"api_load_balancer"`
	}

	var config testConfig
	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !config.APILoadBalancer.Enabled {
		t.Errorf("Expected APILoadBalancer.Enabled to be true, got false")
	}

	if config.APILoadBalancer.Hetzner == nil {
		t.Fatalf("Expected Hetzner config to be set, got nil")
	}

	labels := config.APILoadBalancer.Hetzner.Labels
	if len(labels) != 2 {
		t.Fatalf("Expected 2 labels, got %d", len(labels))
	}

	expectedLabels := map[string]string{
		"cluster_id":  "123456",
		"environment": "production",
	}

	for _, label := range labels {
		expectedValue, exists := expectedLabels[label.Key]
		if !exists {
			t.Errorf("Unexpected label key: %s", label.Key)
			continue
		}
		if label.Value != expectedValue {
			t.Errorf("Expected label %s to have value %s, got %s", label.Key, expectedValue, label.Value)
		}
	}
}

func TestAPILoadBalancer_SetDefaults(t *testing.T) {
	tests := []struct {
		name string
		lb   APILoadBalancer
	}{
		{
			name: "Empty API load balancer",
			lb:   APILoadBalancer{},
		},
		{
			name: "API load balancer with enabled",
			lb: APILoadBalancer{
				Enabled: true,
			},
		},
		{
			name: "API load balancer with labels",
			lb: APILoadBalancer{
				Enabled: true,
				Hetzner: &HetznerConfig{
					Labels: []Label{
						{Key: "test", Value: "value"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := tt.lb
			// SetDefaults should not panic and should preserve values
			lb.SetDefaults()

			// Verify enabled state is preserved
			if lb.Enabled != tt.lb.Enabled {
				t.Errorf("Enabled state changed after SetDefaults")
			}

			// Verify Hetzner config is preserved
			if tt.lb.Hetzner != nil {
				if lb.Hetzner == nil {
					t.Errorf("Hetzner config was cleared after SetDefaults")
				} else if len(lb.Hetzner.Labels) != len(tt.lb.Hetzner.Labels) {
					t.Errorf("Hetzner labels count changed after SetDefaults")
				}
			}
		})
	}
}

func TestAPILoadBalancer_Disabled(t *testing.T) {
	yamlData := `
api_load_balancer:
  enabled: false
`
	type testConfig struct {
		APILoadBalancer APILoadBalancer `yaml:"api_load_balancer"`
	}

	var config testConfig
	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if config.APILoadBalancer.Enabled {
		t.Errorf("Expected APILoadBalancer.Enabled to be false, got true")
	}
}

func TestAPILoadBalancer_EmptyConfig(t *testing.T) {
	yamlData := `
api_load_balancer:
`
	type testConfig struct {
		APILoadBalancer APILoadBalancer `yaml:"api_load_balancer"`
	}

	var config testConfig
	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Empty config should have enabled as false (default)
	if config.APILoadBalancer.Enabled {
		t.Errorf("Expected APILoadBalancer.Enabled to be false by default, got true")
	}
}
