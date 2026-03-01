package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestTargetPools_UnmarshalYAML_StringFormat(t *testing.T) {
	yamlData := `target_pools: "varnish"`

	type testConfig struct {
		TargetPools TargetPools `yaml:"target_pools"`
	}

	var config testConfig
	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(config.TargetPools) != 1 {
		t.Errorf("Expected 1 target pool, got %d", len(config.TargetPools))
	}
	if config.TargetPools[0] != "varnish" {
		t.Errorf("Expected target pool 'varnish', got '%s'", config.TargetPools[0])
	}
}

func TestTargetPools_UnmarshalYAML_ArrayFormat(t *testing.T) {
	yamlData := `target_pools: ["varnish", "nginx"]`

	type testConfig struct {
		TargetPools TargetPools `yaml:"target_pools"`
	}

	var config testConfig
	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(config.TargetPools) != 2 {
		t.Errorf("Expected 2 target pools, got %d", len(config.TargetPools))
	}
	if config.TargetPools[0] != "varnish" {
		t.Errorf("Expected first target pool 'varnish', got '%s'", config.TargetPools[0])
	}
	if config.TargetPools[1] != "nginx" {
		t.Errorf("Expected second target pool 'nginx', got '%s'", config.TargetPools[1])
	}
}

func TestTargetPools_UnmarshalYAML_ArrayFormatMultiline(t *testing.T) {
	yamlData := `target_pools:
  - varnish
  - nginx
  - apache`

	type testConfig struct {
		TargetPools TargetPools `yaml:"target_pools"`
	}

	var config testConfig
	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(config.TargetPools) != 3 {
		t.Errorf("Expected 3 target pools, got %d", len(config.TargetPools))
	}
	expected := []string{"varnish", "nginx", "apache"}
	for i, pool := range config.TargetPools {
		if pool != expected[i] {
			t.Errorf("Expected target pool[%d] '%s', got '%s'", i, expected[i], pool)
		}
	}
}

func TestLoadBalancerSetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		lb       LoadBalancer
		expected LoadBalancer
	}{
		{
			name: "Set defaults for empty load balancer",
			lb: LoadBalancer{
				Enabled: true,
			},
			expected: LoadBalancer{
				Enabled: true,
				Type:    "lb11",
				Algorithm: LoadBalancerAlgorithm{
					Type: "round_robin",
				},
				Services: []LoadBalancerService{
					{
						Protocol:        "tcp",
						ListenPort:      80,
						DestinationPort: 80,
						HealthCheck: &LoadBalancerHealthCheck{
							Protocol: "tcp",
							Port:     80,
							Interval: 15,
							Timeout:  10,
							Retries:  3,
						},
					},
					{
						Protocol:        "tcp",
						ListenPort:      443,
						DestinationPort: 443,
						HealthCheck: &LoadBalancerHealthCheck{
							Protocol: "tcp",
							Port:     443,
							Interval: 15,
							Timeout:  10,
							Retries:  3,
						},
					},
				},
			},
		},
		{
			name: "Keep custom type and algorithm",
			lb: LoadBalancer{
				Enabled: true,
				Type:    "lb31",
				Algorithm: LoadBalancerAlgorithm{
					Type: "least_connections",
				},
			},
			expected: LoadBalancer{
				Enabled: true,
				Type:    "lb31",
				Algorithm: LoadBalancerAlgorithm{
					Type: "least_connections",
				},
				Services: []LoadBalancerService{
					{
						Protocol:        "tcp",
						ListenPort:      80,
						DestinationPort: 80,
						HealthCheck: &LoadBalancerHealthCheck{
							Protocol: "tcp",
							Port:     80,
							Interval: 15,
							Timeout:  10,
							Retries:  3,
						},
					},
					{
						Protocol:        "tcp",
						ListenPort:      443,
						DestinationPort: 443,
						HealthCheck: &LoadBalancerHealthCheck{
							Protocol: "tcp",
							Port:     443,
							Interval: 15,
							Timeout:  10,
							Retries:  3,
						},
					},
				},
			},
		},
		{
			name: "Keep existing services",
			lb: LoadBalancer{
				Enabled: true,
				Services: []LoadBalancerService{
					{
						Protocol:        "tcp",
						ListenPort:      8080,
						DestinationPort: 8080,
					},
				},
			},
			expected: LoadBalancer{
				Enabled: true,
				Type:    "lb11",
				Algorithm: LoadBalancerAlgorithm{
					Type: "round_robin",
				},
				Services: []LoadBalancerService{
					{
						Protocol:        "tcp",
						ListenPort:      8080,
						DestinationPort: 8080,
					},
				},
			},
		},
		{
			name: "Disabled load balancer does not set default services",
			lb: LoadBalancer{
				Enabled: false,
			},
			expected: LoadBalancer{
				Enabled: false,
				Type:    "lb11",
				Algorithm: LoadBalancerAlgorithm{
					Type: "round_robin",
				},
				Services: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := tt.lb
			lb.SetDefaults()

			if lb.Type != tt.expected.Type {
				t.Errorf("Type = %v, want %v", lb.Type, tt.expected.Type)
			}
			if lb.Algorithm.Type != tt.expected.Algorithm.Type {
				t.Errorf("Algorithm.Type = %v, want %v", lb.Algorithm.Type, tt.expected.Algorithm.Type)
			}
			if len(lb.Services) != len(tt.expected.Services) {
				t.Errorf("Services length = %v, want %v", len(lb.Services), len(tt.expected.Services))
			}

			// Compare services
			for i, svc := range lb.Services {
				if i >= len(tt.expected.Services) {
					break
				}
				expectedSvc := tt.expected.Services[i]
				if svc.Protocol != expectedSvc.Protocol {
					t.Errorf("Service[%d].Protocol = %v, want %v", i, svc.Protocol, expectedSvc.Protocol)
				}
				if svc.ListenPort != expectedSvc.ListenPort {
					t.Errorf("Service[%d].ListenPort = %v, want %v", i, svc.ListenPort, expectedSvc.ListenPort)
				}
				if svc.DestinationPort != expectedSvc.DestinationPort {
					t.Errorf("Service[%d].DestinationPort = %v, want %v", i, svc.DestinationPort, expectedSvc.DestinationPort)
				}
			}
		})
	}
}

func TestLoadBalancerHealthCheckDefaults(t *testing.T) {
	lb := LoadBalancer{
		Enabled: true,
	}
	lb.SetDefaults()

	// Check that default services have health checks
	if len(lb.Services) != 2 {
		t.Fatalf("Expected 2 default services, got %d", len(lb.Services))
	}

	for i, svc := range lb.Services {
		if svc.HealthCheck == nil {
			t.Errorf("Service[%d] missing health check", i)
			continue
		}

		hc := svc.HealthCheck
		if hc.Protocol != "tcp" {
			t.Errorf("Service[%d] health check protocol = %v, want tcp", i, hc.Protocol)
		}
		if hc.Interval != 15 {
			t.Errorf("Service[%d] health check interval = %v, want 15", i, hc.Interval)
		}
		if hc.Timeout != 10 {
			t.Errorf("Service[%d] health check timeout = %v, want 10", i, hc.Timeout)
		}
		if hc.Retries != 3 {
			t.Errorf("Service[%d] health check retries = %v, want 3", i, hc.Retries)
		}
	}
}
