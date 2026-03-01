package config

import (
	"strings"
	"testing"
)

func TestValidateLoadBalancer(t *testing.T) {
	tests := []struct {
		name          string
		config        *Main
		expectError   bool
		errorContains string
		warnContains  string
	}{
		{
			name: "Valid load balancer configuration",
			config: &Main{
				ClusterName: "test-cluster",
				LoadBalancer: LoadBalancer{
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
					},
				},
				Networking: Networking{
					PrivateNetwork: PrivateNetwork{
						Enabled: true,
					},
				},
				WorkerNodePools: []WorkerNodePool{
					{
						NodePool: NodePool{
							Name: strPtr("default"),
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Invalid load balancer type",
			config: &Main{
				ClusterName: "test-cluster",
				LoadBalancer: LoadBalancer{
					Enabled: true,
					Type:    "lb99",
					Algorithm: LoadBalancerAlgorithm{
						Type: "round_robin",
					},
				},
			},
			expectError:   true,
			errorContains: "invalid type 'lb99'",
		},
		{
			name: "Invalid algorithm type",
			config: &Main{
				ClusterName: "test-cluster",
				LoadBalancer: LoadBalancer{
					Enabled: true,
					Type:    "lb11",
					Algorithm: LoadBalancerAlgorithm{
						Type: "random",
					},
				},
			},
			expectError:   true,
			errorContains: "invalid algorithm type 'random'",
		},
		{
			name: "Invalid service protocol",
			config: &Main{
				ClusterName: "test-cluster",
				LoadBalancer: LoadBalancer{
					Enabled: true,
					Type:    "lb11",
					Algorithm: LoadBalancerAlgorithm{
						Type: "round_robin",
					},
					Services: []LoadBalancerService{
						{
							Protocol:        "udp",
							ListenPort:      80,
							DestinationPort: 80,
						},
					},
				},
			},
			expectError:   true,
			errorContains: "invalid protocol 'udp'",
		},
		{
			name: "Valid HTTP protocol",
			config: &Main{
				ClusterName: "test-cluster",
				LoadBalancer: LoadBalancer{
					Enabled: true,
					Type:    "lb11",
					Algorithm: LoadBalancerAlgorithm{
						Type: "round_robin",
					},
					Services: []LoadBalancerService{
						{
							Protocol:        "http",
							ListenPort:      80,
							DestinationPort: 80,
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Valid HTTPS protocol",
			config: &Main{
				ClusterName: "test-cluster",
				LoadBalancer: LoadBalancer{
					Enabled: true,
					Type:    "lb11",
					Algorithm: LoadBalancerAlgorithm{
						Type: "round_robin",
					},
					Services: []LoadBalancerService{
						{
							Protocol:        "https",
							ListenPort:      443,
							DestinationPort: 80,
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Invalid listen port",
			config: &Main{
				ClusterName: "test-cluster",
				LoadBalancer: LoadBalancer{
					Enabled: true,
					Type:    "lb11",
					Algorithm: LoadBalancerAlgorithm{
						Type: "round_robin",
					},
					Services: []LoadBalancerService{
						{
							Protocol:        "tcp",
							ListenPort:      99999,
							DestinationPort: 80,
						},
					},
				},
			},
			expectError:   true,
			errorContains: "invalid listen_port",
		},
		{
			name: "Invalid health check interval",
			config: &Main{
				ClusterName: "test-cluster",
				LoadBalancer: LoadBalancer{
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
								Interval: 1,
								Timeout:  10,
								Retries:  3,
							},
						},
					},
				},
			},
			expectError:   true,
			errorContains: "health check interval must be between 3 and 3600",
		},
		{
			name: "Use private IP without private network",
			config: &Main{
				ClusterName: "test-cluster",
				LoadBalancer: LoadBalancer{
					Enabled:      true,
					Type:         "lb11",
					UsePrivateIP: boolPtr(true),
					Algorithm: LoadBalancerAlgorithm{
						Type: "round_robin",
					},
				},
				Networking: Networking{
					PrivateNetwork: PrivateNetwork{
						Enabled: false,
					},
				},
			},
			expectError:   true,
			errorContains: "use_private_ip requires private_network",
		},
		{
			name: "Attach to network without private network",
			config: &Main{
				ClusterName: "test-cluster",
				LoadBalancer: LoadBalancer{
					Enabled:         true,
					Type:            "lb11",
					AttachToNetwork: true,
					Algorithm: LoadBalancerAlgorithm{
						Type: "round_robin",
					},
				},
				Networking: Networking{
					PrivateNetwork: PrivateNetwork{
						Enabled: false,
					},
				},
			},
			expectError:   true,
			errorContains: "attach_to_network requires private_network",
		},
		{
			name: "Target pool not found in worker pools - warning",
			config: &Main{
				ClusterName: "test-cluster",
				LoadBalancer: LoadBalancer{
					Enabled:     true,
					Type:        "lb11",
					TargetPools: []string{"nonexistent"},
					Algorithm: LoadBalancerAlgorithm{
						Type: "round_robin",
					},
				},
				WorkerNodePools: []WorkerNodePool{
					{
						NodePool: NodePool{
							Name: strPtr("default"),
						},
					},
				},
			},
			expectError:  false,
			warnContains: "target_pool 'nonexistent' not found",
		},
		{
			name: "Disabled load balancer - no validation",
			config: &Main{
				ClusterName: "test-cluster",
				LoadBalancer: LoadBalancer{
					Enabled: false,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator(tt.config)
			validator.validateLoadBalancer()

			hasErrors := len(validator.GetErrors()) > 0
			if hasErrors != tt.expectError {
				t.Errorf("Expected error: %v, got errors: %v", tt.expectError, validator.GetErrors())
			}

			if tt.expectError && tt.errorContains != "" {
				found := false
				for _, err := range validator.GetErrors() {
					if strings.Contains(err, tt.errorContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing '%s', got errors: %v", tt.errorContains, validator.GetErrors())
				}
			}

			if tt.warnContains != "" {
				found := false
				for _, warn := range validator.GetWarnings() {
					if strings.Contains(warn, tt.warnContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected warning containing '%s', got warnings: %v", tt.warnContains, validator.GetWarnings())
				}
			}
		})
	}
}

// Helper function for string pointers
func strPtr(s string) *string {
	return &s
}

// Helper function for bool pointers
func boolPtr(b bool) *bool {
	return &b
}
