package addons

import (
	"context"
	"fmt"
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/internal/util"
)

// CiliumInstaller handles Cilium CNI installation using Cilium CLI
type CiliumInstaller struct {
	Config        *config.Main
	SSHClient     *util.SSH
	KubectlClient *util.KubectlClient
	ctx           context.Context
}

// NewCiliumInstaller creates a new Cilium installer
func NewCiliumInstaller(cfg *config.Main, sshClient *util.SSH) *CiliumInstaller {
	return &CiliumInstaller{
		Config:        cfg,
		SSHClient:     sshClient,
		KubectlClient: util.NewKubectlClient(cfg.KubeconfigPath),
		ctx:           context.Background(),
	}
}

// Install installs Cilium CNI using Cilium CLI
func (c *CiliumInstaller) Install(firstMaster *hcloud.Server, masterSSHIP string) error {
	if c.Config.Networking.CNI.Mode != "cilium" {
		return nil // Not using Cilium, skip installation
	}

	if c.Config.Networking.CNI.Cilium == nil {
		return fmt.Errorf("Cilium configuration is missing")
	}

	// Check if Cilium is already installed using cilium CLI
	if c.isCiliumInstalled() {
		util.LogInfo("Cilium CNI already installed, skipping installation", "addons")
		return nil
	}

	util.LogInfo("Installing Cilium CNI", "cilium")

	// Install Cilium using the cilium CLI with local kubeconfig
	if err := c.installCiliumCLI(); err != nil {
		return fmt.Errorf("failed to install Cilium: %w", err)
	}

	// Wait for Cilium to be ready
	if err := c.waitForCiliumReady(); err != nil {
		return fmt.Errorf("failed to verify Cilium status: %w", err)
	}

	util.LogSuccess("Cilium CNI installed successfully", "cilium")
	return nil
}

// isCiliumInstalled checks if Cilium is already installed
func (c *CiliumInstaller) isCiliumInstalled() bool {
	// Check if Cilium daemonset exists in kube-system namespace
	return c.KubectlClient.ResourceExists("daemonset", "cilium", "kube-system")
}

// installCiliumCLI installs Cilium using the cilium CLI tool
func (c *CiliumInstaller) installCiliumCLI() error {
	ciliumConfig := c.Config.Networking.CNI.Cilium

	// Ensure defaults are set
	ciliumConfig.SetDefaults()

	// Expand kubeconfig path to handle tilde (~)
	kubeconfigPath, err := config.ExpandPath(c.Config.KubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to expand kubeconfig path: %w", err)
	}

	// Build the cilium install command with all configuration parameters
	args := []string{
		"install",
	}

	// Add version if specified (use Version field, not HelmChartVersion)
	if ciliumConfig.Version != "" {
		args = append(args, "--version", ciliumConfig.Version)
	}

	// Set IPAM operator cluster pool IPv4 CIDR
	if c.Config.Networking.ClusterCIDR != "" {
		args = append(args, "--set", fmt.Sprintf("ipam.operator.clusterPoolIPv4PodCIDRList=%s", c.Config.Networking.ClusterCIDR))
	}

	// Configure encryption
	if ciliumConfig.EncryptionType != "" {
		switch ciliumConfig.EncryptionType {
		case "wireguard":
			args = append(args, "--set", "encryption.enabled=true")
			args = append(args, "--set", "encryption.type=wireguard")
		case "ipsec":
			args = append(args, "--set", "encryption.enabled=true")
			args = append(args, "--set", "encryption.type=ipsec")
		}
	}

	// Configure routing mode
	if ciliumConfig.RoutingMode != "" {
		args = append(args, "--set", fmt.Sprintf("routingMode=%s", ciliumConfig.RoutingMode))
	}

	// Configure tunnel protocol
	if ciliumConfig.TunnelProtocol != "" {
		args = append(args, "--set", fmt.Sprintf("tunnelProtocol=%s", ciliumConfig.TunnelProtocol))
	}

	// Configure Hubble
	if ciliumConfig.HubbleEnabled != nil && *ciliumConfig.HubbleEnabled {
		args = append(args, "--set", "hubble.enabled=true")

		if ciliumConfig.HubbleRelayEnabled != nil && *ciliumConfig.HubbleRelayEnabled {
			args = append(args, "--set", "hubble.relay.enabled=true")
		}

		if ciliumConfig.HubbleUIEnabled != nil && *ciliumConfig.HubbleUIEnabled {
			args = append(args, "--set", "hubble.ui.enabled=true")
		}

		// Configure Hubble metrics - use enabledList with array format
		if len(ciliumConfig.HubbleMetrics) > 0 {
			// Convert metrics array to comma-separated string format for Helm
			metricsStr := fmt.Sprintf("{%s}", strings.Join(ciliumConfig.HubbleMetrics, ","))
			args = append(args, "--set", fmt.Sprintf("hubble.metrics.enabledList=%s", metricsStr))
		}
	}

	// Configure K8s API server endpoint
	if ciliumConfig.K8sServiceHost != "" {
		args = append(args, "--set", fmt.Sprintf("k8sServiceHost=%s", ciliumConfig.K8sServiceHost))
	}
	if ciliumConfig.K8sServicePort != 0 {
		args = append(args, "--set", fmt.Sprintf("k8sServicePort=%d", ciliumConfig.K8sServicePort))
	}

	// Configure operator replicas
	if ciliumConfig.OperatorReplicas != 0 {
		args = append(args, "--set", fmt.Sprintf("operator.replicas=%d", ciliumConfig.OperatorReplicas))
	}

	// Configure resource requests
	if ciliumConfig.OperatorMemoryRequest != "" {
		args = append(args, "--set", fmt.Sprintf("operator.resources.requests.memory=%s", ciliumConfig.OperatorMemoryRequest))
	}
	if ciliumConfig.AgentMemoryRequest != "" {
		args = append(args, "--set", fmt.Sprintf("resources.requests.memory=%s", ciliumConfig.AgentMemoryRequest))
	}

	// Configure egress gateway
	if ciliumConfig.EgressGatewayEnabled {
		args = append(args, "--set", "egressGateway.enabled=true")
	}

	// Set kubeconfig explicitly with expanded path
	args = append(args, "--kubeconfig", kubeconfigPath)

	// Execute cilium install command with prefix to capture all output
	shell := util.NewShell()
	if err := shell.RunWithPrefix("cilium", "cilium", args...); err != nil {
		return fmt.Errorf("cilium install failed: %w", err)
	}

	util.LogInfo("Cilium installation command completed", "cilium")
	return nil
}

// waitForCiliumReady waits for Cilium to be ready using cilium CLI
func (c *CiliumInstaller) waitForCiliumReady() error {
	util.LogInfo("Waiting for Cilium to be ready...", "cilium")

	// Expand kubeconfig path to handle tilde (~)
	kubeconfigPath, err := config.ExpandPath(c.Config.KubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to expand kubeconfig path: %w", err)
	}

	// Use cilium status --wait to wait for Cilium to be ready with filtered output
	// We only want to show informative messages, not all the detailed status output
	filterFunc := func(line string) bool {
		// Keep only informative messages like version info
		// Discard all the detailed status output (DaemonSet, Deployment, Containers, Errors, Warnings, etc.)
		if strings.Contains(line, "ℹ️") || strings.Contains(line, "Using Cilium version") {
			return true
		}
		// Discard all other lines (status, errors, warnings, pod details, etc.)
		return false
	}

	shell := util.NewShell()
	if err := shell.RunWithFilteredPrefix("cilium", "cilium", filterFunc, "status", "--wait", "--kubeconfig", kubeconfigPath); err != nil {
		return fmt.Errorf("Cilium status check failed: %w", err)
	}

	util.LogSuccess("Cilium is ready", "cilium")
	return nil
}
