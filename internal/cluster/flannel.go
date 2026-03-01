package cluster

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/magenx/kuberaptor/internal/config"
)

const (
	// networkInterfacePlaceholder is used in install scripts for dynamic interface detection
	networkInterfacePlaceholder = "$NETWORK_INTERFACE"
)

// generateFlannelBackendFlags generates flannel backend flags based on configuration
func generateFlannelBackendFlags(cfg *config.Main, k3sVersion string) (string, error) {
	// If CNI is not enabled, return empty
	if !cfg.Networking.CNI.Enabled {
		return "", nil
	}

	// If using Flannel CNI
	if cfg.Networking.CNI.Mode == "flannel" {
		// Check if encryption is enabled (default is true)
		useEncryption := true
		if cfg.Networking.CNI.Flannel != nil {
			useEncryption = cfg.Networking.CNI.Flannel.IsEncryptionEnabled()
		}

		if useEncryption {
			// Determine which wireguard backend to use based on k3s version
			useNativeWireguard, err := shouldUseNativeWireguard(k3sVersion)
			if err != nil {
				return "", err
			}

			if useNativeWireguard {
				return "--flannel-backend=wireguard-native", nil
			}
			return "--flannel-backend=wireguard", nil
		}
		// No encryption, use default flannel backend (vxlan)
		return "", nil
	}

	// Using a different CNI (e.g., Cilium)
	args := []string{"--flannel-backend=none", "--disable-network-policy"}

	// Check if we should disable kube-proxy
	// For Cilium, kube-proxy is typically disabled
	// For Flannel with disable_kube_proxy=true, also disable it
	if cfg.Networking.CNI.Mode == "cilium" {
		args = append(args, "--disable-kube-proxy")
	} else if cfg.Networking.CNI.Flannel != nil && cfg.Networking.CNI.Flannel.DisableKubeProxy {
		args = append(args, "--disable-kube-proxy")
	}

	return strings.Join(args, " "), nil
}

// shouldUseNativeWireguard determines if wireguard-native backend should be used
// wireguard-native is available in k3s >= v1.23.6+k3s1
func shouldUseNativeWireguard(k3sVersion string) (bool, error) {
	// Simple version comparison for v1.23.6+k3s1 and later
	// The wireguard-native backend was introduced in v1.23.6+k3s1

	// Extract major, minor, patch from version string like "v1.24.0+k3s1" or "v1.23.6+k3s1"
	version := strings.TrimPrefix(k3sVersion, "v")
	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return false, fmt.Errorf("invalid k3s version format: %s", k3sVersion)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false, fmt.Errorf("invalid major version: %s", parts[0])
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	// Extract patch number (handle +k3s1 suffix)
	patchParts := strings.Split(parts[2], "+")
	patch, err := strconv.Atoi(patchParts[0])
	if err != nil {
		return false, fmt.Errorf("invalid patch version: %s", patchParts[0])
	}

	// Check if version >= 1.23.6
	if major > 1 {
		return true, nil
	}
	if major == 1 {
		if minor > 23 {
			return true, nil
		}
		if minor == 23 && patch >= 6 {
			return true, nil
		}
	}

	return false, nil
}

// generateFlannelIfaceFlags generates flannel interface flags for private networks
// This is used when private network is enabled to specify the interface flannel should use
func generateFlannelIfaceFlags(cfg *config.Main) string {
	// Only add flannel-iface if:
	// 1. CNI is enabled
	// 2. Using flannel mode
	// 3. Private network is enabled
	if cfg.Networking.CNI.Enabled &&
		cfg.Networking.CNI.Mode == "flannel" &&
		cfg.Networking.PrivateNetwork.Enabled {
		// The actual interface will be detected at runtime in the install script
		// This is a placeholder that will be replaced with the actual interface
		return "--flannel-iface=" + networkInterfacePlaceholder
	}
	return ""
}

// shouldConfigureFlannelInterface checks if flannel interface should be configured
func shouldConfigureFlannelInterface(cfg *config.Main) bool {
	return cfg.Networking.PrivateNetwork.Enabled &&
		cfg.Networking.CNI.Enabled &&
		cfg.Networking.CNI.Mode == "flannel"
}
