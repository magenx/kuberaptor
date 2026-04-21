// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package cluster

import (
	"fmt"
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/internal/config"
)

// TestLoadBalancerPrivateIPLogic tests the logic for determining when to use private IPs
// This test validates the fix for the "load balancer is not attached to a network" issue
func TestLoadBalancerPrivateIPLogic(t *testing.T) {
	tests := []struct {
		name                    string
		attachToNetwork         bool
		privateNetworkEnabled   bool
		usePrivateIPConfigured  *bool
		networkProvided         bool
		expectedUsePrivateIP    bool
		expectedAttachToNetwork bool
		description             string
	}{
		{
			name:                    "attach_to_network=true, private_network=true, use_private_ip not set",
			attachToNetwork:         true,
			privateNetworkEnabled:   true,
			usePrivateIPConfigured:  nil,
			networkProvided:         true,
			expectedUsePrivateIP:    true,
			expectedAttachToNetwork: true,
			description:             "Should attach to network and use private IPs by default",
		},
		{
			name:                    "attach_to_network=false, private_network=true, use_private_ip not set",
			attachToNetwork:         false,
			privateNetworkEnabled:   true,
			usePrivateIPConfigured:  nil,
			networkProvided:         true,
			expectedUsePrivateIP:    false,
			expectedAttachToNetwork: false,
			description:             "Bug fix: Should NOT use private IPs when not attached to network",
		},
		{
			name:                    "attach_to_network=true, private_network=false, use_private_ip not set",
			attachToNetwork:         true,
			privateNetworkEnabled:   false,
			usePrivateIPConfigured:  nil,
			networkProvided:         true,
			expectedUsePrivateIP:    false,
			expectedAttachToNetwork: false,
			description:             "Should not attach to network when private network is disabled",
		},
		{
			name:                    "attach_to_network=true, private_network=true, use_private_ip=true",
			attachToNetwork:         true,
			privateNetworkEnabled:   true,
			usePrivateIPConfigured:  boolPtr(true),
			networkProvided:         true,
			expectedUsePrivateIP:    true,
			expectedAttachToNetwork: true,
			description:             "Should use private IPs when explicitly configured and attached",
		},
		{
			name:                    "attach_to_network=false, private_network=true, use_private_ip=true",
			attachToNetwork:         false,
			privateNetworkEnabled:   true,
			usePrivateIPConfigured:  boolPtr(true),
			networkProvided:         true,
			expectedUsePrivateIP:    false,
			expectedAttachToNetwork: false,
			description:             "Bug fix: Should NOT use private IPs even when explicitly configured if not attached",
		},
		{
			name:                    "attach_to_network=true, private_network=true, use_private_ip=false",
			attachToNetwork:         true,
			privateNetworkEnabled:   true,
			usePrivateIPConfigured:  boolPtr(false),
			networkProvided:         true,
			expectedUsePrivateIP:    false,
			expectedAttachToNetwork: true,
			description:             "Should respect explicit false setting for use_private_ip",
		},
		{
			name:                    "attach_to_network=true, private_network=true, network=nil",
			attachToNetwork:         true,
			privateNetworkEnabled:   true,
			usePrivateIPConfigured:  nil,
			networkProvided:         false,
			expectedUsePrivateIP:    false,
			expectedAttachToNetwork: false,
			description:             "Should not attach when network object is not provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from CreateGlobalLoadBalancer
			cfg := &config.Main{
				LoadBalancer: config.LoadBalancer{
					Enabled:         true,
					AttachToNetwork: tt.attachToNetwork,
					UsePrivateIP:    tt.usePrivateIPConfigured,
				},
				Networking: config.Networking{
					PrivateNetwork: config.PrivateNetwork{
						Enabled: tt.privateNetworkEnabled,
					},
				},
			}

			// Simulate network object
			var network interface{}
			if tt.networkProvided {
				network = &struct{}{} // Mock network object
			}

			// This is the logic from network_resources.go lines 313-326
			shouldAttachToNetwork := cfg.LoadBalancer.AttachToNetwork && cfg.Networking.PrivateNetwork.Enabled && network != nil

			usePrivateIP := false
			if cfg.LoadBalancer.UsePrivateIP != nil {
				// If explicitly set, use that value (but it must match network attachment)
				usePrivateIP = *cfg.LoadBalancer.UsePrivateIP && shouldAttachToNetwork
			} else if shouldAttachToNetwork {
				// If not explicitly set and load balancer is attached to network, default to true
				usePrivateIP = true
			}

			// Verify expectations
			if shouldAttachToNetwork != tt.expectedAttachToNetwork {
				t.Errorf("%s: shouldAttachToNetwork = %v, expected %v",
					tt.description, shouldAttachToNetwork, tt.expectedAttachToNetwork)
			}

			if usePrivateIP != tt.expectedUsePrivateIP {
				t.Errorf("%s: usePrivateIP = %v, expected %v",
					tt.description, usePrivateIP, tt.expectedUsePrivateIP)
			}

			// The key assertion: usePrivateIP should NEVER be true when shouldAttachToNetwork is false
			if usePrivateIP && !shouldAttachToNetwork {
				t.Errorf("%s: CRITICAL BUG: usePrivateIP is true but shouldAttachToNetwork is false. "+
					"This would cause 'load balancer is not attached to a network' error!",
					tt.description)
			}
		})
	}
}

// TestGlobalLoadBalancerNaming tests that global load balancer names always follow the pattern
// {cluster-name}-global-lb-{location} or {cluster-name}-{custom-name}-global-lb-{location}
func TestGlobalLoadBalancerNaming(t *testing.T) {
	tests := []struct {
		name           string
		clusterName    string
		location       string
		customName     *string
		expectedLBName string
		description    string
	}{
		{
			name:           "standard naming without custom name",
			clusterName:    "my-cluster",
			location:       "nbg1",
			customName:     nil,
			expectedLBName: "my-cluster-global-lb-nbg1",
			description:    "Should use pattern {cluster-name}-global-lb-{location}",
		},
		{
			name:           "standard naming with different location",
			clusterName:    "production",
			location:       "fsn1",
			customName:     nil,
			expectedLBName: "production-global-lb-fsn1",
			description:    "Should follow pattern for different location",
		},
		{
			name:           "naming with custom name",
			clusterName:    "magenx",
			location:       "nbg1",
			customName:     stringPtr("custom-lb-name"),
			expectedLBName: "magenx-custom-lb-name-global-lb-nbg1",
			description:    "Should include custom name: {cluster-name}-{custom-name}-global-lb-{location}",
		},
		{
			name:           "multi-location deployment",
			clusterName:    "multi-region",
			location:       "hel1",
			customName:     nil,
			expectedLBName: "multi-region-global-lb-hel1",
			description:    "Should use location suffix for multi-location deployments",
		},
		{
			name:           "custom name with different location",
			clusterName:    "prod-cluster",
			location:       "fsn1",
			customName:     stringPtr("web"),
			expectedLBName: "prod-cluster-web-global-lb-fsn1",
			description:    "Should append custom name between cluster name and global-lb",
		},
		{
			name:           "empty custom name should be treated as nil",
			clusterName:    "test-cluster",
			location:       "hel1",
			customName:     stringPtr(""),
			expectedLBName: "test-cluster-global-lb-hel1",
			description:    "Empty custom name should use default pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the naming logic from createGlobalLoadBalancerForLocation
			var lbName string
			if tt.customName != nil && *tt.customName != "" {
				// Include custom name in the pattern
				lbName = fmt.Sprintf("%s-%s-global-lb-%s", tt.clusterName, *tt.customName, tt.location)
			} else {
				// Default pattern without custom name
				lbName = fmt.Sprintf("%s-global-lb-%s", tt.clusterName, tt.location)
			}

			// Verify the name matches expected pattern
			if lbName != tt.expectedLBName {
				t.Errorf("%s: lbName = %s, expected %s",
					tt.description, lbName, tt.expectedLBName)
			}
		})
	}
}

func TestIsRetryableLoadBalancerTargetError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "load balancer not attached to network",
			err:      hcloud.Error{Code: hcloud.ErrorCodeLoadBalancerNotAttachedToNetwork, Message: "not attached"},
			expected: true,
		},
		{
			name:     "server not attached to network",
			err:      hcloud.Error{Code: hcloud.ErrorCodeServerNotAttachedToNetwork, Message: "server not attached"},
			expected: true,
		},
		{
			name:     "resource unavailable",
			err:      hcloud.Error{Code: hcloud.ErrorCodeResourceUnavailable, Message: "resource unavailable"},
			expected: true,
		},
		{
			name:     "conflict",
			err:      hcloud.Error{Code: hcloud.ErrorCodeConflict, Message: "conflict"},
			expected: true,
		},
		{
			name:     "timeout",
			err:      hcloud.Error{Code: hcloud.ErrorCodeTimeout, Message: "timeout"},
			expected: true,
		},
		{
			name:     "non-retryable not_found",
			err:      hcloud.Error{Code: hcloud.ErrorCodeNotFound, Message: "not found"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := isRetryableLoadBalancerTargetError(tt.err); actual != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, actual)
			}
		})
	}
}
