package cluster

import (
	"net"
	"strings"
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/internal/config"
)

// TestGenerateTLSSans tests the TLS SAN generation function
func TestGenerateTLSSans(t *testing.T) {
	tests := []struct {
		name               string
		cfg                *config.Main
		masters            []*hcloud.Server
		expectedContains   []string
		expectedNotContain []string
	}{
		{
			name: "single master with public IP only",
			cfg: &config.Main{
				Networking: config.Networking{
					PrivateNetwork: config.PrivateNetwork{
						Enabled: false,
					},
				},
			},
			masters: []*hcloud.Server{
				{
					Name: "master-1",
					PublicNet: hcloud.ServerPublicNet{
						IPv4: hcloud.ServerPublicNetIPv4{
							IP: net.ParseIP("46.224.204.161"),
						},
					},
				},
			},
			expectedContains: []string{
				"--tls-san=46.224.204.161",
				"--tls-san=127.0.0.1",
			},
		},
		{
			name: "single master with private and public IPs",
			cfg: &config.Main{
				Networking: config.Networking{
					PrivateNetwork: config.PrivateNetwork{
						Enabled: true,
						Subnet:  "10.0.0.0/16",
					},
				},
			},
			masters: []*hcloud.Server{
				{
					Name: "master-1",
					PublicNet: hcloud.ServerPublicNet{
						IPv4: hcloud.ServerPublicNetIPv4{
							IP: net.ParseIP("46.224.204.161"),
						},
					},
					PrivateNet: []hcloud.ServerPrivateNet{
						{
							IP: net.ParseIP("10.0.0.2"),
						},
					},
				},
			},
			expectedContains: []string{
				"--tls-san=10.0.0.2",
				"--tls-san=46.224.204.161",
				"--tls-san=127.0.0.1",
			},
		},
		{
			name: "multiple masters with private and public IPs",
			cfg: &config.Main{
				Networking: config.Networking{
					PrivateNetwork: config.PrivateNetwork{
						Enabled: true,
						Subnet:  "10.0.0.0/16",
					},
				},
			},
			masters: []*hcloud.Server{
				{
					Name: "master-1",
					PublicNet: hcloud.ServerPublicNet{
						IPv4: hcloud.ServerPublicNetIPv4{
							IP: net.ParseIP("46.224.204.161"),
						},
					},
					PrivateNet: []hcloud.ServerPrivateNet{
						{
							IP: net.ParseIP("10.0.0.2"),
						},
					},
				},
				{
					Name: "master-2",
					PublicNet: hcloud.ServerPublicNet{
						IPv4: hcloud.ServerPublicNetIPv4{
							IP: net.ParseIP("46.224.204.162"),
						},
					},
					PrivateNet: []hcloud.ServerPrivateNet{
						{
							IP: net.ParseIP("10.0.0.3"),
						},
					},
				},
			},
			expectedContains: []string{
				"--tls-san=10.0.0.2",
				"--tls-san=10.0.0.3",
				"--tls-san=46.224.204.161",
				"--tls-san=46.224.204.162",
				"--tls-san=127.0.0.1",
			},
		},
		{
			name: "with API server hostname",
			cfg: &config.Main{
				APIServerHostname: "k8s.example.com",
				Networking: config.Networking{
					PrivateNetwork: config.PrivateNetwork{
						Enabled: false,
					},
				},
			},
			masters: []*hcloud.Server{
				{
					Name: "master-1",
					PublicNet: hcloud.ServerPublicNet{
						IPv4: hcloud.ServerPublicNetIPv4{
							IP: net.ParseIP("46.224.204.161"),
						},
					},
				},
			},
			expectedContains: []string{
				"--tls-san=46.224.204.161",
				"--tls-san=127.0.0.1",
				"--tls-san=k8s.example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateTLSSans(tt.cfg, tt.masters, tt.masters[0], nil)
			if err != nil {
				t.Fatalf("GenerateTLSSans() error = %v", err)
			}

			// Check that all expected strings are present
			for _, expected := range tt.expectedContains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain '%s', but got: %s", expected, result)
				}
			}

			// Check that unwanted strings are not present
			for _, notExpected := range tt.expectedNotContain {
				if strings.Contains(result, notExpected) {
					t.Errorf("Expected result NOT to contain '%s', but got: %s", notExpected, result)
				}
			}

			// Verify no duplicates by splitting and checking
			parts := strings.Fields(result)
			seen := make(map[string]bool)
			for _, part := range parts {
				if seen[part] {
					t.Errorf("Found duplicate TLS SAN: %s", part)
				}
				seen[part] = true
			}
		})
	}
}

// TestGenerateTLSSansWithAPILoadBalancer tests TLS SAN generation with API load balancer
func TestGenerateTLSSansWithAPILoadBalancer(t *testing.T) {
	cfg := &config.Main{
		Networking: config.Networking{
			PrivateNetwork: config.PrivateNetwork{
				Enabled: true,
				Subnet:  "10.0.0.0/16",
			},
		},
	}

	masters := []*hcloud.Server{
		{
			Name: "master-1",
			PublicNet: hcloud.ServerPublicNet{
				IPv4: hcloud.ServerPublicNetIPv4{
					IP: net.ParseIP("46.224.204.161"),
				},
			},
			PrivateNet: []hcloud.ServerPrivateNet{
				{
					IP: net.ParseIP("10.0.0.2"),
				},
			},
		},
	}

	apiLB := &hcloud.LoadBalancer{
		Name: "api-lb",
		PublicNet: hcloud.LoadBalancerPublicNet{
			IPv4: hcloud.LoadBalancerPublicNetIPv4{
				IP: net.ParseIP("162.55.155.23"),
			},
		},
	}

	result, err := GenerateTLSSans(cfg, masters, masters[0], []*hcloud.LoadBalancer{apiLB})
	if err != nil {
		t.Fatalf("GenerateTLSSans() error = %v", err)
	}

	expectedContains := []string{
		"--tls-san=10.0.0.2",
		"--tls-san=46.224.204.161",
		"--tls-san=127.0.0.1",
		"--tls-san=162.55.155.23", // API load balancer IP
	}

	for _, expected := range expectedContains {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected result to contain '%s', but got: %s", expected, result)
		}
	}

	// Verify no duplicates
	parts := strings.Fields(result)
	seen := make(map[string]bool)
	for _, part := range parts {
		if seen[part] {
			t.Errorf("Found duplicate TLS SAN: %s", part)
		}
		seen[part] = true
	}
}
