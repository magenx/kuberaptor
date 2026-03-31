// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package cluster

import (
	"testing"

	"github.com/magenx/kuberaptor/internal/config"
)

func TestGenerateFlannelBackendFlags(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config.Main
		k3sVersion    string
		expectedFlags string
		expectError   bool
	}{
		{
			name: "flannel with encryption and new k3s version",
			cfg: &config.Main{
				Networking: config.Networking{
					CNI: config.CNI{
						Enabled: true,
						Mode:    "flannel",
					},
				},
			},
			k3sVersion:    "v1.24.0+k3s1",
			expectedFlags: "--flannel-backend=wireguard-native",
			expectError:   false,
		},
		{
			name: "flannel with encryption and old k3s version",
			cfg: &config.Main{
				Networking: config.Networking{
					CNI: config.CNI{
						Enabled: true,
						Mode:    "flannel",
					},
				},
			},
			k3sVersion:    "v1.23.5+k3s1",
			expectedFlags: "--flannel-backend=wireguard",
			expectError:   false,
		},
		{
			name: "flannel with exact version v1.23.6+k3s1",
			cfg: &config.Main{
				Networking: config.Networking{
					CNI: config.CNI{
						Enabled: true,
						Mode:    "flannel",
					},
				},
			},
			k3sVersion:    "v1.23.6+k3s1",
			expectedFlags: "--flannel-backend=wireguard-native",
			expectError:   false,
		},
		{
			name: "cilium mode",
			cfg: &config.Main{
				Networking: config.Networking{
					CNI: config.CNI{
						Enabled: true,
						Mode:    "cilium",
					},
				},
			},
			k3sVersion:    "v1.24.0+k3s1",
			expectedFlags: "--flannel-backend=none --disable-network-policy --disable-kube-proxy",
			expectError:   false,
		},
		{
			name: "non-flannel CNI with disable_kube_proxy",
			cfg: &config.Main{
				Networking: config.Networking{
					CNI: config.CNI{
						Enabled: true,
						Mode:    "calico",
						Flannel: &config.Flannel{
							DisableKubeProxy: true,
						},
					},
				},
			},
			k3sVersion:    "v1.24.0+k3s1",
			expectedFlags: "--flannel-backend=none --disable-network-policy --disable-kube-proxy",
			expectError:   false,
		},
		{
			name: "flannel with encryption disabled",
			cfg: &config.Main{
				Networking: config.Networking{
					CNI: config.CNI{
						Enabled: true,
						Mode:    "flannel",
						Flannel: &config.Flannel{
							Encryption: boolPtr(false),
						},
					},
				},
			},
			k3sVersion:    "v1.24.0+k3s1",
			expectedFlags: "",
			expectError:   false,
		},
		{
			name: "cni disabled",
			cfg: &config.Main{
				Networking: config.Networking{
					CNI: config.CNI{
						Enabled: false,
					},
				},
			},
			k3sVersion:    "v1.24.0+k3s1",
			expectedFlags: "",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags, err := generateFlannelBackendFlags(tt.cfg, tt.k3sVersion)
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if flags != tt.expectedFlags {
				t.Errorf("expected flags %q, got %q", tt.expectedFlags, flags)
			}
		})
	}
}

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}

func TestShouldUseNativeWireguard(t *testing.T) {
	tests := []struct {
		name        string
		k3sVersion  string
		expected    bool
		expectError bool
	}{
		{
			name:        "version 1.24.0",
			k3sVersion:  "v1.24.0+k3s1",
			expected:    true,
			expectError: false,
		},
		{
			name:        "version 1.23.6",
			k3sVersion:  "v1.23.6+k3s1",
			expected:    true,
			expectError: false,
		},
		{
			name:        "version 1.23.5",
			k3sVersion:  "v1.23.5+k3s1",
			expected:    false,
			expectError: false,
		},
		{
			name:        "version 1.22.0",
			k3sVersion:  "v1.22.0+k3s1",
			expected:    false,
			expectError: false,
		},
		{
			name:        "version 1.25.0",
			k3sVersion:  "v1.25.0+k3s1",
			expected:    true,
			expectError: false,
		},
		{
			name:        "version 2.0.0",
			k3sVersion:  "v2.0.0+k3s1",
			expected:    true,
			expectError: false,
		},
		{
			name:        "invalid version format",
			k3sVersion:  "invalid",
			expected:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := shouldUseNativeWireguard(tt.k3sVersion)
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGenerateFlannelIfaceFlags(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Main
		expected string
	}{
		{
			name: "private network with flannel",
			cfg: &config.Main{
				Networking: config.Networking{
					CNI: config.CNI{
						Enabled: true,
						Mode:    "flannel",
					},
					PrivateNetwork: config.PrivateNetwork{
						Enabled: true,
					},
				},
			},
			expected: "--flannel-iface=$NETWORK_INTERFACE",
		},
		{
			name: "private network with cilium",
			cfg: &config.Main{
				Networking: config.Networking{
					CNI: config.CNI{
						Enabled: true,
						Mode:    "cilium",
					},
					PrivateNetwork: config.PrivateNetwork{
						Enabled: true,
					},
				},
			},
			expected: "",
		},
		{
			name: "public network with flannel",
			cfg: &config.Main{
				Networking: config.Networking{
					CNI: config.CNI{
						Enabled: true,
						Mode:    "flannel",
					},
					PrivateNetwork: config.PrivateNetwork{
						Enabled: false,
					},
				},
			},
			expected: "",
		},
		{
			name: "cni disabled",
			cfg: &config.Main{
				Networking: config.Networking{
					CNI: config.CNI{
						Enabled: false,
					},
					PrivateNetwork: config.PrivateNetwork{
						Enabled: true,
					},
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateFlannelIfaceFlags(tt.cfg)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
