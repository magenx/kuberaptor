// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package util

import "github.com/magenx/kuberaptor/internal/config"

// ResolveNetworkName determines the network name based on configuration
// Returns empty string if private network is disabled, otherwise returns the
// configured existing network name, or the cluster name when no existing network is specified
func ResolveNetworkName(cfg *config.Main) string {
	// If private network is not enabled, return empty string
	if !cfg.Networking.PrivateNetwork.Enabled {
		return ""
	}

	// If an existing network name is configured, use it
	if cfg.Networking.PrivateNetwork.ExistingNetworkName != "" {
		return cfg.Networking.PrivateNetwork.ExistingNetworkName
	}

	// Otherwise, use the cluster name (which matches the network creation logic)
	return cfg.ClusterName
}
