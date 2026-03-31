// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package cluster

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/internal/util"
)

const (
	// HCloudNodeGroupLabel is the label key used by the cluster autoscaler to identify node groups
	HCloudNodeGroupLabel = "hcloud/node-group"
)

// serverLister is an interface for listing Hetzner Cloud servers.
type serverLister interface {
	ListServers(context.Context, hcloud.ServerListOpts) ([]*hcloud.Server, error)
}

// GetServerIP returns the appropriate IP address for a server based on network configuration
// It prefers private IP if private networking is enabled and available, otherwise uses public IPv4
func GetServerIP(server *hcloud.Server, cfg *config.Main) (string, error) {
	// Prefer private IP if private networking is enabled and available
	if cfg.Networking.PrivateNetwork.Enabled && len(server.PrivateNet) > 0 {
		return server.PrivateNet[0].IP.String(), nil
	}

	// Fall back to public IPv4
	if server.PublicNet.IPv4.IP != nil {
		return server.PublicNet.IPv4.IP.String(), nil
	}

	return "", fmt.Errorf("server %s has no accessible IP address (private networking disabled or unavailable, and no public IPv4)", server.Name)
}

// GetServerPublicIP returns the public IPv4 address of a server for external access (e.g., kubeconfig)
func GetServerPublicIP(server *hcloud.Server) (string, error) {
	if server.PublicNet.IPv4.IP != nil {
		return server.PublicNet.IPv4.IP.String(), nil
	}
	return "", fmt.Errorf("server %s has no public IPv4 address", server.Name)
}

// GetServerSSHIP returns the appropriate IP address for SSH connections from external machines
// It always prefers public IP for SSH access, as the control machine may not have access to private IPs.
// The fallback to private IP is only for edge cases where servers are created without public IPs
// (e.g., when public_network.ipv4 is explicitly disabled in the configuration).
func GetServerSSHIP(server *hcloud.Server) (string, error) {
	// Always prefer public IP for SSH from external machines
	if server.PublicNet.IPv4.IP != nil {
		return server.PublicNet.IPv4.IP.String(), nil
	}

	// Fall back to private IP only if no public IP is available
	// Note: SSH will likely fail from external machines in this case unless VPN/bastion is used
	if len(server.PrivateNet) > 0 {
		return server.PrivateNet[0].IP.String(), nil
	}

	return "", fmt.Errorf("server %s has no accessible IP address for SSH", server.Name)
}

// GenerateTLSSans generates TLS SAN (Subject Alternative Name) flags for k3s installation
// This ensures the k3s API server certificate includes all necessary IP addresses and hostnames
func GenerateTLSSans(cfg *config.Main, masters []*hcloud.Server, firstMaster *hcloud.Server, apiLoadBalancers []*hcloud.LoadBalancer) (string, error) {
	// Use a map to collect unique SANs while building the list
	uniqueSans := make(map[string]bool)

	// Add first master's API server IP (private IP if available, otherwise public)
	apiServerIP, err := GetServerIP(firstMaster, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to get API server IP: %w", err)
	}
	uniqueSans[fmt.Sprintf("--tls-san=%s", apiServerIP)] = true

	// Always add localhost
	uniqueSans["--tls-san=127.0.0.1"] = true

	// Add all API load balancer IPs if configured and created
	for _, apiLoadBalancer := range apiLoadBalancers {
		if apiLoadBalancer != nil && apiLoadBalancer.PublicNet.IPv4.IP != nil {
			lbIP := apiLoadBalancer.PublicNet.IPv4.IP.String()
			uniqueSans[fmt.Sprintf("--tls-san=%s", lbIP)] = true
		}
	}

	// Add API server hostname if configured
	if cfg.APIServerHostname != "" {
		uniqueSans[fmt.Sprintf("--tls-san=%s", cfg.APIServerHostname)] = true
	}

	// Add all master IPs (both private and public)
	for _, master := range masters {
		// Add private IP
		if len(master.PrivateNet) > 0 {
			privateIP := master.PrivateNet[0].IP.String()
			uniqueSans[fmt.Sprintf("--tls-san=%s", privateIP)] = true
		}

		// Add public IP
		if master.PublicNet.IPv4.IP != nil {
			publicIP := master.PublicNet.IPv4.IP.String()
			uniqueSans[fmt.Sprintf("--tls-san=%s", publicIP)] = true
		}
	}

	// Convert map to sorted slice
	sortedSans := make([]string, 0, len(uniqueSans))
	for san := range uniqueSans {
		sortedSans = append(sortedSans, san)
	}

	// Sort for deterministic output
	sort.Strings(sortedSans)

	return strings.Join(sortedSans, " "), nil
}

// FindNATGatewayForBastion finds and returns the first NAT gateway server for a cluster
// Returns nil if no NAT gateway is found or if NAT gateway is not enabled
func FindNATGatewayForBastion(ctx context.Context, hetznerClient serverLister, clusterName string) (*hcloud.Server, error) {
	// Find NAT gateway servers by label (there may be multiple gateways, one per location)
	// We use labels instead of name because NAT gateways have location-specific names
	natGatewayLabel := fmt.Sprintf("cluster=%s,role=nat-gateway", clusterName)
	natGateways, err := hetznerClient.ListServers(ctx, hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: natGatewayLabel,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list NAT gateway servers: %w", err)
	}

	if len(natGateways) == 0 {
		// No NAT gateways found
		return nil, nil
	}

	// Use the first NAT gateway as bastion host
	// When multiple gateways exist across locations, the first one is selected.
	// This is sufficient because all nodes in the private network can be reached
	// through any NAT gateway regardless of its location.
	return natGateways[0], nil
}

// findAutoscaledPoolServers finds servers created by the cluster autoscaler.
// These servers have the HCloudNodeGroupLabel label instead of the cluster label.
func findAutoscaledPoolServers(ctx context.Context, cfg *config.Main, hetznerClient serverLister) ([]*hcloud.Server, error) {
	var allServers []*hcloud.Server

	// Iterate through all worker node pools
	for _, pool := range cfg.WorkerNodePools {
		// Only process autoscaling-enabled pools
		if !pool.AutoscalingEnabled() {
			continue
		}

		// Build the node pool name (must match the name used by cluster autoscaler)
		poolName := pool.BuildNodePoolName(cfg.ClusterName)

		// Search for servers with the HCloudNodeGroupLabel
		nodeGroupLabel := fmt.Sprintf("%s=%s", HCloudNodeGroupLabel, poolName)
		servers, err := hetznerClient.ListServers(ctx, hcloud.ServerListOpts{
			ListOpts: hcloud.ListOpts{
				LabelSelector: nodeGroupLabel,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list servers for node group %s: %w", poolName, err)
		}

		allServers = append(allServers, servers...)
	}

	return allServers, nil
}

// configureNATGatewayBastion configures the NAT gateway as a bastion host if enabled.
// The operation parameter is used in log messages (e.g., "run", "upgrade").
func configureNATGatewayBastion(ctx context.Context, cfg *config.Main, hetznerClient serverLister, sshClient *util.SSH, operation string) error {
	// Check if NAT gateway is enabled
	if cfg.Networking.PrivateNetwork.NATGateway == nil ||
		!cfg.Networking.PrivateNetwork.NATGateway.Enabled {
		return nil
	}

	// Find NAT gateway server for use as bastion host
	natGateway, err := FindNATGatewayForBastion(ctx, hetznerClient, cfg.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to find NAT gateway for bastion: %w", err)
	}

	if natGateway == nil {
		// NAT gateway might not exist yet (e.g., cluster not created)
		// This is not an error - just skip bastion configuration
		return nil
	}

	// Get NAT gateway public IP
	bastionIP, err := GetServerPublicIP(natGateway)
	if err != nil {
		return fmt.Errorf("failed to get NAT gateway public IP: %w", err)
	}

	// Configure SSH to use NAT gateway as bastion host
	sshClient.SetBastion(bastionIP, cfg.Networking.SSH.Port)
	util.LogInfo(fmt.Sprintf("Using NAT gateway %s (%s) as SSH bastion host", natGateway.Name, bastionIP), operation)

	return nil
}
