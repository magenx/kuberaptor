// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package cluster

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/internal/util"
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

// TestGetServerPublicIP tests the GetServerPublicIP function
func TestGetServerPublicIP(t *testing.T) {
	t.Run("server with public IPv4", func(t *testing.T) {
		server := &hcloud.Server{
			Name: "test-server",
			PublicNet: hcloud.ServerPublicNet{
				IPv4: hcloud.ServerPublicNetIPv4{
					IP: net.ParseIP("1.2.3.4"),
				},
			},
		}
		ip, err := GetServerPublicIP(server)
		if err != nil {
			t.Fatalf("GetServerPublicIP() error = %v", err)
		}
		if ip != "1.2.3.4" {
			t.Errorf("expected '1.2.3.4', got %q", ip)
		}
	})

	t.Run("server without public IPv4", func(t *testing.T) {
		server := &hcloud.Server{
			Name:      "no-ip-server",
			PublicNet: hcloud.ServerPublicNet{},
		}
		_, err := GetServerPublicIP(server)
		if err == nil {
			t.Error("expected error for server without public IPv4, got nil")
		}
	})
}

// TestGetServerSSHIP tests the GetServerSSHIP function
func TestGetServerSSHIP(t *testing.T) {
	t.Run("server with public IPv4 preferred", func(t *testing.T) {
		server := &hcloud.Server{
			Name: "test-server",
			PublicNet: hcloud.ServerPublicNet{
				IPv4: hcloud.ServerPublicNetIPv4{
					IP: net.ParseIP("5.6.7.8"),
				},
			},
			PrivateNet: []hcloud.ServerPrivateNet{
				{IP: net.ParseIP("10.0.0.2")},
			},
		}
		ip, err := GetServerSSHIP(server)
		if err != nil {
			t.Fatalf("GetServerSSHIP() error = %v", err)
		}
		// Public IP should be preferred for SSH
		if ip != "5.6.7.8" {
			t.Errorf("expected public IP '5.6.7.8', got %q", ip)
		}
	})

	t.Run("server with only private IP falls back", func(t *testing.T) {
		server := &hcloud.Server{
			Name:      "private-only",
			PublicNet: hcloud.ServerPublicNet{},
			PrivateNet: []hcloud.ServerPrivateNet{
				{IP: net.ParseIP("10.0.0.5")},
			},
		}
		ip, err := GetServerSSHIP(server)
		if err != nil {
			t.Fatalf("GetServerSSHIP() error = %v", err)
		}
		if ip != "10.0.0.5" {
			t.Errorf("expected fallback private IP '10.0.0.5', got %q", ip)
		}
	})

	t.Run("server with no accessible IP", func(t *testing.T) {
		server := &hcloud.Server{
			Name:      "no-ip-server",
			PublicNet: hcloud.ServerPublicNet{},
		}
		_, err := GetServerSSHIP(server)
		if err == nil {
			t.Error("expected error for server with no accessible IP, got nil")
		}
	})
}

// mockServerLister is a mock implementation of the serverLister interface
type mockServerLister struct {
	servers []*hcloud.Server
	err     error
}

func (m *mockServerLister) ListServers(_ context.Context, _ hcloud.ServerListOpts) ([]*hcloud.Server, error) {
	return m.servers, m.err
}

// TestFindNATGatewayForBastion tests the FindNATGatewayForBastion function
func TestFindNATGatewayForBastion(t *testing.T) {
	ctx := context.Background()

	t.Run("no NAT gateways found returns nil", func(t *testing.T) {
		lister := &mockServerLister{servers: []*hcloud.Server{}}

		gw, err := FindNATGatewayForBastion(ctx, lister, "my-cluster")
		if err != nil {
			t.Fatalf("FindNATGatewayForBastion() error = %v", err)
		}
		if gw != nil {
			t.Errorf("expected nil gateway, got %v", gw)
		}
	})

	t.Run("returns first NAT gateway found", func(t *testing.T) {
		gw1 := &hcloud.Server{ID: 1, Name: "nat-gw-fsn1"}
		gw2 := &hcloud.Server{ID: 2, Name: "nat-gw-nbg1"}
		lister := &mockServerLister{servers: []*hcloud.Server{gw1, gw2}}

		gw, err := FindNATGatewayForBastion(ctx, lister, "my-cluster")
		if err != nil {
			t.Fatalf("FindNATGatewayForBastion() error = %v", err)
		}
		if gw == nil {
			t.Fatal("expected non-nil gateway")
		}
		if gw.ID != 1 {
			t.Errorf("expected first gateway (ID=1), got ID=%d", gw.ID)
		}
	})

	t.Run("lister error is propagated", func(t *testing.T) {
		lister := &mockServerLister{err: fmt.Errorf("API error")}

		_, err := FindNATGatewayForBastion(ctx, lister, "my-cluster")
		if err == nil {
			t.Error("expected error from lister, got nil")
		}
	})
}

// TestFindAutoscaledPoolServersHelper tests the findAutoscaledPoolServers helper function in helpers.go
func TestFindAutoscaledPoolServersHelper(t *testing.T) {
	ctx := context.Background()

	t.Run("no autoscaling pools returns empty", func(t *testing.T) {
		cfg := &config.Main{
			ClusterName: "test-cluster",
			WorkerNodePools: []config.WorkerNodePool{
				{
					NodePool: config.NodePool{
						InstanceType:  "cx22",
						InstanceCount: 2,
					},
					Locations: []string{"fsn1"},
				},
			},
		}
		lister := &mockServerLister{servers: []*hcloud.Server{}}

		servers, err := findAutoscaledPoolServers(ctx, cfg, lister)
		if err != nil {
			t.Fatalf("findAutoscaledPoolServers() error = %v", err)
		}
		if len(servers) != 0 {
			t.Errorf("expected 0 servers, got %d", len(servers))
		}
	})

	t.Run("autoscaling pool queries lister", func(t *testing.T) {
		s1 := &hcloud.Server{ID: 10, Name: "autoscale-server-1"}
		cfg := &config.Main{
			ClusterName: "test-cluster",
			WorkerNodePools: []config.WorkerNodePool{
				{
					NodePool: config.NodePool{
						InstanceType: "cx22",
						Autoscaling: &config.Autoscaling{
							Enabled:      true,
							MinInstances: 1,
							MaxInstances: 5,
						},
					},
					Locations: []string{"fsn1"},
				},
			},
		}
		lister := &mockServerLister{servers: []*hcloud.Server{s1}}

		servers, err := findAutoscaledPoolServers(ctx, cfg, lister)
		if err != nil {
			t.Fatalf("findAutoscaledPoolServers() error = %v", err)
		}
		if len(servers) != 1 {
			t.Errorf("expected 1 server, got %d", len(servers))
		}
	})

	t.Run("lister error is propagated", func(t *testing.T) {
		cfg := &config.Main{
			ClusterName: "test-cluster",
			WorkerNodePools: []config.WorkerNodePool{
				{
					NodePool: config.NodePool{
						InstanceType: "cx22",
						Autoscaling: &config.Autoscaling{
							Enabled:      true,
							MinInstances: 1,
							MaxInstances: 3,
						},
					},
					Locations: []string{"fsn1"},
				},
			},
		}
		lister := &mockServerLister{err: fmt.Errorf("API unavailable")}

		_, err := findAutoscaledPoolServers(ctx, cfg, lister)
		if err == nil {
			t.Error("expected error from lister, got nil")
		}
	})
}

// TestConfigureNATGatewayBastion_Disabled tests that configureNATGatewayBastion
// returns nil when NAT gateway is not enabled.
func TestConfigureNATGatewayBastion_Disabled(t *testing.T) {
	ctx := context.Background()

	t.Run("nil NAT gateway config", func(t *testing.T) {
		cfg := &config.Main{
			ClusterName: "test-cluster",
			Networking: config.Networking{
				PrivateNetwork: config.PrivateNetwork{
					NATGateway: nil,
				},
			},
		}
		lister := &mockServerLister{}
		sshClient := util.NewSSHFromKeys([]byte("priv"), []byte("pub"))

		err := configureNATGatewayBastion(ctx, cfg, lister, sshClient, "test")
		if err != nil {
			t.Errorf("expected nil error when NAT gateway not configured, got %v", err)
		}
	})

	t.Run("NAT gateway disabled", func(t *testing.T) {
		cfg := &config.Main{
			ClusterName: "test-cluster",
			Networking: config.Networking{
				PrivateNetwork: config.PrivateNetwork{
					NATGateway: &config.NATGateway{
						Enabled: false,
					},
				},
			},
		}
		lister := &mockServerLister{}
		sshClient := util.NewSSHFromKeys([]byte("priv"), []byte("pub"))

		err := configureNATGatewayBastion(ctx, cfg, lister, sshClient, "test")
		if err != nil {
			t.Errorf("expected nil error when NAT gateway disabled, got %v", err)
		}
	})
}

// TestConfigureNATGatewayBastion_NoGatewayFound tests that when no NAT gateway
// server exists, the function returns nil (not an error).
func TestConfigureNATGatewayBastion_NoGatewayFound(t *testing.T) {
	ctx := context.Background()

	cfg := &config.Main{
		ClusterName: "test-cluster",
		Networking: config.Networking{
			PrivateNetwork: config.PrivateNetwork{
				NATGateway: &config.NATGateway{
					Enabled: true,
				},
			},
		},
	}
	lister := &mockServerLister{servers: []*hcloud.Server{}}
	sshClient := util.NewSSHFromKeys([]byte("priv"), []byte("pub"))

	err := configureNATGatewayBastion(ctx, cfg, lister, sshClient, "test")
	if err != nil {
		t.Errorf("expected nil error when no NAT gateway servers found, got %v", err)
	}
}

// TestConfigureNATGatewayBastion_SetsBastion tests that when a NAT gateway with
// a public IP is found, the bastion is configured on the SSH client.
func TestConfigureNATGatewayBastion_SetsBastion(t *testing.T) {
	ctx := context.Background()

	cfg := &config.Main{
		ClusterName: "test-cluster",
		Networking: config.Networking{
			SSH: config.SSH{
				Port: 22,
			},
			PrivateNetwork: config.PrivateNetwork{
				NATGateway: &config.NATGateway{
					Enabled: true,
				},
			},
		},
	}
	natGW := &hcloud.Server{
		ID:   100,
		Name: "nat-gw-fsn1",
		PublicNet: hcloud.ServerPublicNet{
			IPv4: hcloud.ServerPublicNetIPv4{
				IP: net.ParseIP("45.11.22.33"),
			},
		},
	}
	lister := &mockServerLister{servers: []*hcloud.Server{natGW}}
	sshClient := util.NewSSHFromKeys([]byte("priv"), []byte("pub"))

	err := configureNATGatewayBastion(ctx, cfg, lister, sshClient, "test")
	if err != nil {
		t.Fatalf("configureNATGatewayBastion() error = %v", err)
	}

	// Verify that bastion was configured on the SSH client
	if sshClient.BastionHost() != "45.11.22.33" {
		t.Errorf("expected bastion host '45.11.22.33', got %q", sshClient.BastionHost())
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
