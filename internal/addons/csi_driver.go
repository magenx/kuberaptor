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

// CSIDriverInstaller installs the Hetzner CSI driver
type CSIDriverInstaller struct {
	Config        *config.Main
	SSHClient     *util.SSH
	KubectlClient *util.KubectlClient
	ctx           context.Context
}

// NewCSIDriverInstaller creates a new CSI driver installer
func NewCSIDriverInstaller(cfg *config.Main, sshClient *util.SSH) *CSIDriverInstaller {
	return &CSIDriverInstaller{
		Config:        cfg,
		SSHClient:     sshClient,
		KubectlClient: util.NewKubectlClient(cfg.KubeconfigPath),
		ctx:           context.Background(),
	}
}

// Install installs the CSI driver using local kubectl with kubeconfig
func (c *CSIDriverInstaller) Install(firstMaster *hcloud.Server, masterIP string) error {
	// Check if CSI driver is already installed
	if c.KubectlClient.ResourceExists("daemonset", "hcloud-csi-node", "kube-system") &&
		c.KubectlClient.ResourceExists("statefulset", "hcloud-csi-controller", "kube-system") {
		util.LogInfo("Hetzner CSI driver already installed, skipping installation", "addons")
		return nil
	}

	// Create Hetzner secret first using local kubectl
	if err := c.createHetznerSecret(); err != nil {
		return fmt.Errorf("failed to create Hetzner secret: %w", err)
	}

	// Apply CSI driver manifest directly from URL using local kubectl
	// No patching needed - the manifest works as-is when applied via the Kubernetes API
	manifestURL := c.Config.Addons.CSIDriver.ManifestURL
	if err := c.KubectlClient.Apply(manifestURL); err != nil {
		return fmt.Errorf("failed to apply CSI driver manifest: %w", err)
	}

	util.LogSuccess("Hetzner CSI driver installed", "addons")
	return nil
}

// createHetznerSecret creates the Hetzner Cloud secret required by CSI driver and CCM
func (c *CSIDriverInstaller) createHetznerSecret() error {
	// Resolve network name based on configuration using shared utility
	networkName := util.ResolveNetworkName(c.Config)

	// Create secret manifest
	secretManifest := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: hcloud
  namespace: kube-system
stringData:
  token: %s
  network: %s
`, c.Config.HetznerToken, networkName)

	// Apply secret using local kubectl
	return c.KubectlClient.ApplyManifest(secretManifest)
}
