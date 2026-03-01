package addons

import (
	"context"
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/internal/util"
)

// SystemUpgradeControllerInstaller installs the system upgrade controller
type SystemUpgradeControllerInstaller struct {
	Config        *config.Main
	SSHClient     *util.SSH
	KubectlClient *util.KubectlClient
	ctx           context.Context
}

// NewSystemUpgradeControllerInstaller creates a new system upgrade controller installer
func NewSystemUpgradeControllerInstaller(cfg *config.Main, sshClient *util.SSH) *SystemUpgradeControllerInstaller {
	return &SystemUpgradeControllerInstaller{
		Config:        cfg,
		SSHClient:     sshClient,
		KubectlClient: util.NewKubectlClient(cfg.KubeconfigPath),
		ctx:           context.Background(),
	}
}

// Install installs the system upgrade controller using local kubectl
func (s *SystemUpgradeControllerInstaller) Install(firstMaster *hcloud.Server, masterIP string) error {
	// Check if system upgrade controller is already installed
	if s.KubectlClient.ResourceExists("deployment", "system-upgrade-controller", "system-upgrade") {
		util.LogInfo("System upgrade controller already installed, skipping installation", "addons")
		return nil
	}

	// Install CRDs first
	if s.Config.Addons.SystemUpgradeController.CRDManifestURL != "" {
		if err := s.KubectlClient.Apply(s.Config.Addons.SystemUpgradeController.CRDManifestURL); err != nil {
			return fmt.Errorf("failed to apply system upgrade controller CRDs: %w", err)
		}
	}

	// Install deployment
	if err := s.KubectlClient.Apply(s.Config.Addons.SystemUpgradeController.DeploymentManifestURL); err != nil {
		return fmt.Errorf("failed to apply system upgrade controller deployment: %w", err)
	}

	util.LogSuccess("System upgrade controller installed", "addons")
	return nil
}
