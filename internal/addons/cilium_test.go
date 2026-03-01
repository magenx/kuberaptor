package addons

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/magenx/kuberaptor/internal/config"
)

func TestCiliumInstaller_Creation(t *testing.T) {
	cfg := &config.Main{
		KubeconfigPath: "~/.kube/config",
		HetznerToken:   "test-token",
		ClusterName:    "test-cluster",
		Networking: config.Networking{
			CNI: config.CNI{
				Mode: "cilium",
				Cilium: &config.Cilium{
					Enabled: true,
				},
			},
		},
	}

	// Ensure defaults are set
	cfg.Networking.CNI.Cilium.SetDefaults()

	installer := NewCiliumInstaller(cfg, nil)

	if installer == nil {
		t.Error("Expected non-nil installer")
	}
	if installer.Config != cfg {
		t.Error("Expected config to be set")
	}
}

func TestCiliumInstaller_DefaultValues(t *testing.T) {
	cilium := &config.Cilium{}
	cilium.SetDefaults()

	// Check that defaults are properly set
	if cilium.Version != "1.17.2" {
		t.Errorf("Expected Version to be '1.17.2', got '%s'", cilium.Version)
	}
	if cilium.EncryptionType != "wireguard" {
		t.Errorf("Expected EncryptionType to be 'wireguard', got '%s'", cilium.EncryptionType)
	}
	if cilium.RoutingMode != "tunnel" {
		t.Errorf("Expected RoutingMode to be 'tunnel', got '%s'", cilium.RoutingMode)
	}
	if cilium.TunnelProtocol != "vxlan" {
		t.Errorf("Expected TunnelProtocol to be 'vxlan', got '%s'", cilium.TunnelProtocol)
	}
	if cilium.HubbleEnabled == nil || !*cilium.HubbleEnabled {
		t.Error("Expected HubbleEnabled to be true")
	}
	if cilium.HubbleRelayEnabled == nil || !*cilium.HubbleRelayEnabled {
		t.Error("Expected HubbleRelayEnabled to be true")
	}
	if cilium.HubbleUIEnabled == nil || !*cilium.HubbleUIEnabled {
		t.Error("Expected HubbleUIEnabled to be true")
	}
	if cilium.K8sServiceHost != "127.0.0.1" {
		t.Errorf("Expected K8sServiceHost to be '127.0.0.1', got '%s'", cilium.K8sServiceHost)
	}
	if cilium.K8sServicePort != 6444 {
		t.Errorf("Expected K8sServicePort to be 6444, got %d", cilium.K8sServicePort)
	}
	if cilium.OperatorReplicas != 1 {
		t.Errorf("Expected OperatorReplicas to be 1, got %d", cilium.OperatorReplicas)
	}
	if cilium.OperatorMemoryRequest != "128Mi" {
		t.Errorf("Expected OperatorMemoryRequest to be '128Mi', got '%s'", cilium.OperatorMemoryRequest)
	}
	if cilium.AgentMemoryRequest != "512Mi" {
		t.Errorf("Expected AgentMemoryRequest to be '512Mi', got '%s'", cilium.AgentMemoryRequest)
	}
}

func TestCiliumInstaller_PathExpansion(t *testing.T) {
	// Test that tilde paths are properly expanded
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name         string
		inputPath    string
		expectedPath string
	}{
		{
			name:         "tilde path expansion",
			inputPath:    "~/.kube/config",
			expectedPath: filepath.Join(homeDir, ".kube/config"),
		},
		{
			name:         "absolute path unchanged",
			inputPath:    "/etc/rancher/k3s/k3s.yaml",
			expectedPath: "/etc/rancher/k3s/k3s.yaml",
		},
		{
			name:         "tilde with subdirectory",
			inputPath:    "~/my-configs/kubeconfig.yaml",
			expectedPath: filepath.Join(homeDir, "my-configs/kubeconfig.yaml"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expandedPath, err := config.ExpandPath(tt.inputPath)
			if err != nil {
				t.Fatalf("ExpandPath failed: %v", err)
			}

			// Make the expected path absolute for comparison
			expectedAbs, err := filepath.Abs(tt.expectedPath)
			if err != nil {
				t.Fatalf("Failed to get absolute path: %v", err)
			}

			if expandedPath != expectedAbs {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.inputPath, expandedPath, expectedAbs)
			}

			// Verify that tilde is not in the expanded path
			if strings.Contains(expandedPath, "~") {
				t.Errorf("Expanded path still contains tilde: %q", expandedPath)
			}
		})
	}
}
