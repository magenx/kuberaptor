package util

import (
	"testing"

	"github.com/magenx/kuberaptor/internal/config"
)

func TestResolveNetworkName(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Main
		expected string
	}{
		{
			name: "private network disabled",
			config: &config.Main{
				ClusterName: "test-cluster",
				Networking: config.Networking{
					PrivateNetwork: config.PrivateNetwork{
						Enabled: false,
					},
				},
			},
			expected: "",
		},
		{
			name: "private network enabled with existing network",
			config: &config.Main{
				ClusterName: "test-cluster",
				Networking: config.Networking{
					PrivateNetwork: config.PrivateNetwork{
						Enabled:             true,
						ExistingNetworkName: "my-existing-network",
					},
				},
			},
			expected: "my-existing-network",
		},
		{
			name: "private network enabled without existing network",
			config: &config.Main{
				ClusterName: "test-cluster",
				Networking: config.Networking{
					PrivateNetwork: config.PrivateNetwork{
						Enabled:             true,
						ExistingNetworkName: "",
					},
				},
			},
			expected: "test-cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveNetworkName(tt.config)
			if result != tt.expected {
				t.Errorf("ResolveNetworkName() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
