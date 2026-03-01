package addons

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/internal/util"
)

// CloudControllerManagerInstaller installs the Hetzner cloud controller manager
type CloudControllerManagerInstaller struct {
	Config        *config.Main
	SSHClient     *util.SSH
	KubectlClient *util.KubectlClient
	ctx           context.Context
}

// NewCloudControllerManagerInstaller creates a new cloud controller manager installer
func NewCloudControllerManagerInstaller(cfg *config.Main, sshClient *util.SSH) *CloudControllerManagerInstaller {
	return &CloudControllerManagerInstaller{
		Config:        cfg,
		SSHClient:     sshClient,
		KubectlClient: util.NewKubectlClient(cfg.KubeconfigPath),
		ctx:           context.Background(),
	}
}

// Install installs the cloud controller manager using local kubectl
func (c *CloudControllerManagerInstaller) Install(firstMaster *hcloud.Server, masterIP string) error {
	// Check if cloud controller manager is already installed
	if c.KubectlClient.ResourceExists("deployment", "hcloud-cloud-controller-manager", "kube-system") {
		util.LogInfo("Hetzner cloud controller manager already installed, skipping installation", "addons")
		return nil
	}

	// Download and patch the manifest
	manifestURL := c.resolveManifestURL()
	manifest, err := c.fetchManifest(manifestURL)
	if err != nil {
		return fmt.Errorf("failed to fetch cloud controller manager manifest: %w", err)
	}

	// Patch the manifest for configuration (not K3s-specific, but for cluster config)
	manifest = c.patchClusterCIDR(manifest)
	manifest = c.patchSecurePort(manifest)

	// Apply using local kubectl
	if err := c.KubectlClient.ApplyManifest(manifest); err != nil {
		return fmt.Errorf("failed to apply cloud controller manager manifest: %w", err)
	}

	util.LogSuccess("Hetzner cloud controller manager installed", "addons")
	return nil
}

// resolveManifestURL determines the correct manifest URL based on network configuration
func (c *CloudControllerManagerInstaller) resolveManifestURL() string {
	baseURL := c.Config.Addons.CloudControllerManager.ManifestURL

	// If private network is not enabled, use the non-networks version
	if !c.Config.Networking.PrivateNetwork.Enabled {
		return strings.ReplaceAll(baseURL, "-networks", "")
	}

	return baseURL
}

// fetchManifest downloads the manifest from the given URL
func (c *CloudControllerManagerInstaller) fetchManifest(url string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download manifest: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read manifest: %w", err)
	}

	return string(body), nil
}

// patchClusterCIDR patches the cluster CIDR in the manifest
func (c *CloudControllerManagerInstaller) patchClusterCIDR(manifest string) string {
	clusterCIDR := c.Config.Networking.ClusterCIDR
	re := regexp.MustCompile(`--cluster-cidr=[^\s"]+`)
	return re.ReplaceAllString(manifest, fmt.Sprintf("--cluster-cidr=%s", clusterCIDR))
}

// patchSecurePort adds the --secure-port=0 flag to prevent port 10258 binding conflict
// Note: This uses string replacement which depends on the manifest format remaining stable.
// The alternative would be to parse the YAML, modify it, and re-serialize, but that would
// add complexity and dependencies. This approach works reliably with the current manifest format.
func (c *CloudControllerManagerInstaller) patchSecurePort(manifest string) string {
	// Add --secure-port=0 after --webhook-secure-port=0
	// This prevents the cloud controller manager from binding to port 10258 on the host network
	return strings.ReplaceAll(manifest,
		`- "--webhook-secure-port=0"`,
		"- \"--webhook-secure-port=0\"\n            - \"--secure-port=0\"")
}
