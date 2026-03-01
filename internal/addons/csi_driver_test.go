package addons

import (
	"testing"

	"github.com/magenx/kuberaptor/internal/config"
)

func TestCSIDriverInstaller_Creation(t *testing.T) {
	cfg := &config.Main{
		KubeconfigPath: "~/.kube/config",
		HetznerToken:   "test-token",
		ClusterName:    "test-cluster",
	}
	installer := NewCSIDriverInstaller(cfg, nil)

	if installer == nil {
		t.Error("Expected non-nil installer")
	}
	if installer.Config != cfg {
		t.Error("Expected config to be set")
	}
}
