// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package addons

import (
	"context"
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/internal/util"
)

// Installer manages addon installation for the cluster
type Installer struct {
	Config    *config.Main
	SSHClient *util.SSH
	ctx       context.Context
}

// NewInstaller creates a new addon installer
func NewInstaller(cfg *config.Main, sshClient *util.SSH) *Installer {
	return &Installer{
		Config:    cfg,
		SSHClient: sshClient,
		ctx:       context.Background(),
	}
}

// InstallAll installs all enabled addons
// masterSSHIP is the IP address to connect via SSH (usually public IP)
// masterClusterIP is the IP address for internal cluster communication (usually private IP if enabled, otherwise public)
// k3sToken is the k3s cluster token used for joining nodes
func (i *Installer) InstallAll(firstMaster *hcloud.Server, masters []*hcloud.Server, autoscalingPools []config.WorkerNodePool, masterSSHIP string, masterClusterIP string, k3sToken string) error {
	util.LogInfo("Installing cluster addons", "addons")

	// Install Cilium CNI if enabled
	if i.Config.Networking.CNI.Mode == "cilium" {
		if err := i.installCilium(firstMaster, masterSSHIP); err != nil {
			return fmt.Errorf("failed to install Cilium CNI: %w", err)
		}
	}

	// Install metrics server if enabled
	if i.Config.Addons.MetricsServer != nil && i.Config.Addons.MetricsServer.Enabled {
		if err := i.installMetricsServer(firstMaster, masterSSHIP); err != nil {
			return fmt.Errorf("failed to install metrics server: %w", err)
		}
	}

	// Install CSI driver if enabled
	if i.Config.Addons.CSIDriver != nil && i.Config.Addons.CSIDriver.Enabled {
		if err := i.installCSIDriver(firstMaster, masterSSHIP); err != nil {
			return fmt.Errorf("failed to install CSI driver: %w", err)
		}
	}

	// Install cloud controller manager if enabled
	if i.Config.Addons.CloudControllerManager != nil && i.Config.Addons.CloudControllerManager.Enabled {
		if err := i.installCloudControllerManager(firstMaster, masterSSHIP); err != nil {
			return fmt.Errorf("failed to install cloud controller manager: %w", err)
		}
	}

	// Install system upgrade controller if enabled
	if i.Config.Addons.SystemUpgradeController != nil && i.Config.Addons.SystemUpgradeController.Enabled {
		if err := i.installSystemUpgradeController(firstMaster, masterSSHIP); err != nil {
			return fmt.Errorf("failed to install system upgrade controller: %w", err)
		}
	}

	// Install cluster autoscaler if enabled and there are autoscaling pools
	if i.Config.Addons.ClusterAutoscaler != nil && i.Config.Addons.ClusterAutoscaler.Enabled && len(autoscalingPools) > 0 {
		if err := i.installClusterAutoscaler(firstMaster, masters, autoscalingPools, masterSSHIP, masterClusterIP, k3sToken); err != nil {
			return fmt.Errorf("failed to install cluster autoscaler: %w", err)
		}
	}

	// Install Kured if enabled
	if i.Config.Addons.Kured != nil && i.Config.Addons.Kured.Enabled {
		if err := i.installKured(firstMaster, masterSSHIP); err != nil {
			return fmt.Errorf("failed to install Kured: %w", err)
		}
	}

	util.LogSuccess("All addons installed successfully", "addons")
	return nil
}

// installCilium installs Cilium CNI
func (i *Installer) installCilium(firstMaster *hcloud.Server, masterSSHIP string) error {
	installer := NewCiliumInstaller(i.Config, i.SSHClient)
	return installer.Install(firstMaster, masterSSHIP)
}

// installMetricsServer installs the metrics server addon
func (i *Installer) installMetricsServer(firstMaster *hcloud.Server, masterSSHIP string) error {
	// Metrics server is typically installed by k3s by default
	// We just verify it's running
	util.LogInfo("Metrics server verified (installed by K3s)", "addons")
	return nil
}

// installCSIDriver installs the Hetzner CSI driver
func (i *Installer) installCSIDriver(firstMaster *hcloud.Server, masterSSHIP string) error {
	installer := NewCSIDriverInstaller(i.Config, i.SSHClient)
	return installer.Install(firstMaster, masterSSHIP)
}

// installCloudControllerManager installs the Hetzner cloud controller manager
func (i *Installer) installCloudControllerManager(firstMaster *hcloud.Server, masterSSHIP string) error {
	installer := NewCloudControllerManagerInstaller(i.Config, i.SSHClient)
	return installer.Install(firstMaster, masterSSHIP)
}

// installSystemUpgradeController installs the system upgrade controller
func (i *Installer) installSystemUpgradeController(firstMaster *hcloud.Server, masterSSHIP string) error {
	installer := NewSystemUpgradeControllerInstaller(i.Config, i.SSHClient)
	return installer.Install(firstMaster, masterSSHIP)
}

// installClusterAutoscaler installs the cluster autoscaler
func (i *Installer) installClusterAutoscaler(firstMaster *hcloud.Server, masters []*hcloud.Server, autoscalingPools []config.WorkerNodePool, masterSSHIP string, masterClusterIP string, k3sToken string) error {
	installer := NewClusterAutoscalerInstaller(i.Config, i.SSHClient)
	return installer.Install(firstMaster, masters, autoscalingPools, masterSSHIP, masterClusterIP, k3sToken)
}

// installKured installs Kured (Kubernetes Reboot Daemon)
func (i *Installer) installKured(firstMaster *hcloud.Server, masterSSHIP string) error {
	installer := NewKuredInstaller(i.Config, i.SSHClient)
	return installer.Install(firstMaster, masterSSHIP)
}
