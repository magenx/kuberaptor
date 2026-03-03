// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package addons

import (
	"strings"
	"testing"

	"github.com/magenx/kuberaptor/internal/config"
)

func TestPatchSecurePort(t *testing.T) {
	cfg := &config.Main{
		Networking: config.Networking{
			ClusterCIDR: "10.244.0.0/16",
		},
	}

	installer := NewCloudControllerManagerInstaller(cfg, nil)

	// Sample manifest section with webhook-secure-port
	manifest := `          args:
            - "--allow-untagged-cloud"
            - "--cloud-provider=hcloud"
            - "--webhook-secure-port=0"
            - "--allocate-node-cidrs=true"
            - "--cluster-cidr=10.244.0.0/16"`

	patched := installer.patchSecurePort(manifest)

	// Verify that --secure-port=0 was added after --webhook-secure-port=0
	if !strings.Contains(patched, `- "--webhook-secure-port=0"`) {
		t.Error("Expected to find --webhook-secure-port=0")
	}
	if !strings.Contains(patched, `- "--secure-port=0"`) {
		t.Error("Expected to find --secure-port=0")
	}

	// Verify proper placement - secure-port should come after webhook-secure-port
	webhookIdx := strings.Index(patched, `- "--webhook-secure-port=0"`)
	secureIdx := strings.Index(patched, `- "--secure-port=0"`)
	if secureIdx <= webhookIdx {
		t.Error("Expected --secure-port=0 to come after --webhook-secure-port=0")
	}
}

func TestPatchClusterCIDR(t *testing.T) {
	cfg := &config.Main{
		Networking: config.Networking{
			ClusterCIDR: "10.100.0.0/16",
		},
	}

	installer := NewCloudControllerManagerInstaller(cfg, nil)

	manifest := `          args:
            - "--cluster-cidr=10.244.0.0/16"
            - "--allocate-node-cidrs=true"`

	patched := installer.patchClusterCIDR(manifest)

	// Verify that cluster CIDR was replaced with our custom value
	if !strings.Contains(patched, "--cluster-cidr=10.100.0.0/16") {
		t.Error("Expected cluster CIDR to be patched to 10.100.0.0/16")
	}
	if strings.Contains(patched, "--cluster-cidr=10.244.0.0/16") {
		t.Error("Expected old cluster CIDR to be replaced")
	}
}

func TestResolveManifestURL(t *testing.T) {
	tests := []struct {
		name                  string
		privateNetworkEnabled bool
		baseURL               string
		expectedURL           string
	}{
		{
			name:                  "with private network",
			privateNetworkEnabled: true,
			baseURL:               "https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.28.0/ccm-networks.yaml",
			expectedURL:           "https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.28.0/ccm-networks.yaml",
		},
		{
			name:                  "without private network",
			privateNetworkEnabled: false,
			baseURL:               "https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.28.0/ccm-networks.yaml",
			expectedURL:           "https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.28.0/ccm.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Main{
				Networking: config.Networking{
					PrivateNetwork: config.PrivateNetwork{
						Enabled: tt.privateNetworkEnabled,
					},
				},
				Addons: config.Addons{
					CloudControllerManager: &config.CloudControllerManager{
						ManifestURL: tt.baseURL,
					},
				},
			}

			installer := NewCloudControllerManagerInstaller(cfg, nil)
			result := installer.resolveManifestURL()

			if result != tt.expectedURL {
				t.Errorf("Expected URL %s, got %s", tt.expectedURL, result)
			}
		})
	}
}

func TestPatchingOrder(t *testing.T) {
	cfg := &config.Main{
		Networking: config.Networking{
			ClusterCIDR: "10.100.0.0/16",
		},
	}

	installer := NewCloudControllerManagerInstaller(cfg, nil)

	// Full manifest section to test both patches
	manifest := `          args:
            - "--allow-untagged-cloud"
            - "--cloud-provider=hcloud"
            - "--webhook-secure-port=0"
            - "--allocate-node-cidrs=true"
            - "--cluster-cidr=10.244.0.0/16"`

	// Apply both patches
	patched := installer.patchClusterCIDR(manifest)
	patched = installer.patchSecurePort(patched)

	// Verify both patches were applied
	if !strings.Contains(patched, "--cluster-cidr=10.100.0.0/16") {
		t.Error("Expected cluster CIDR to be patched")
	}
	if !strings.Contains(patched, `- "--secure-port=0"`) {
		t.Error("Expected secure-port to be added")
	}
}
