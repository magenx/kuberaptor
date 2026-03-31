// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package addons

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/internal/cloudinit"
	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/internal/util"
	"gopkg.in/yaml.v3"
)

//go:embed templates/worker_install_script.sh
var workerInstallScriptTemplate string

// ClusterAutoscalerInstaller installs the cluster autoscaler
type ClusterAutoscalerInstaller struct {
	Config        *config.Main
	SSHClient     *util.SSH
	KubectlClient *util.KubectlClient
	ctx           context.Context
}

// NewClusterAutoscalerInstaller creates a new cluster autoscaler installer
func NewClusterAutoscalerInstaller(cfg *config.Main, sshClient *util.SSH) *ClusterAutoscalerInstaller {
	return &ClusterAutoscalerInstaller{
		Config:        cfg,
		SSHClient:     sshClient,
		KubectlClient: util.NewKubectlClient(cfg.KubeconfigPath),
		ctx:           context.Background(),
	}
}

// Install installs the cluster autoscaler using local kubectl
func (c *ClusterAutoscalerInstaller) Install(firstMaster *hcloud.Server, masters []*hcloud.Server, autoscalingPools []config.WorkerNodePool, masterSSHIP string, masterClusterIP string, k3sToken string) error {
	// Check if cluster autoscaler is already installed
	if c.KubectlClient.ResourceExists("deployment", "cluster-autoscaler", "kube-system") {
		util.LogInfo("Cluster autoscaler already installed, skipping installation", "addons")
		return nil
	}

	// Fetch and patch the manifest
	manifest, err := c.generateManifest(firstMaster, masters, autoscalingPools, masterClusterIP, k3sToken)
	if err != nil {
		return fmt.Errorf("failed to generate manifest: %w", err)
	}

	// Apply using local kubectl
	if err := c.KubectlClient.ApplyManifest(manifest); err != nil {
		return fmt.Errorf("failed to apply cluster autoscaler manifest: %w", err)
	}

	util.LogSuccess("Cluster autoscaler installed", "addons")
	return nil
}

// generateManifest fetches and patches the cluster autoscaler manifest
func (c *ClusterAutoscalerInstaller) generateManifest(firstMaster *hcloud.Server, masters []*hcloud.Server, autoscalingPools []config.WorkerNodePool, masterClusterIP string, k3sToken string) (string, error) {
	// Fetch the manifest
	manifestURL := c.Config.Addons.ClusterAutoscaler.ManifestURL
	resp, err := http.Get(manifestURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch manifest, status: %d", resp.StatusCode)
	}

	manifestBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read manifest: %w", err)
	}

	// Split manifest into separate resources
	manifestStr := string(manifestBytes)
	resources := strings.Split(manifestStr, "---\n")

	var patchedResources []string
	for _, resource := range resources {
		resource = strings.TrimSpace(resource)
		if resource == "" {
			continue
		}

		// Parse YAML resource
		var doc map[string]interface{}
		if err := yaml.Unmarshal([]byte(resource), &doc); err != nil {
			// If parsing fails, keep original
			patchedResources = append(patchedResources, resource)
			continue
		}

		// Patch based on kind
		kind, _ := doc["kind"].(string)
		switch kind {
		case "Deployment":
			if err := c.patchDeployment(doc, firstMaster, masters, autoscalingPools, masterClusterIP, k3sToken); err != nil {
				return "", err
			}
		case "ClusterRole":
			c.patchClusterRole(doc)
		}

		// Convert back to YAML
		patchedBytes, err := yaml.Marshal(doc)
		if err != nil {
			return "", fmt.Errorf("failed to marshal patched resource: %w", err)
		}
		patchedResources = append(patchedResources, string(patchedBytes))
	}

	return strings.Join(patchedResources, "---\n"), nil
}

// patchDeployment patches the deployment resource
func (c *ClusterAutoscalerInstaller) patchDeployment(doc map[string]interface{}, firstMaster *hcloud.Server, masters []*hcloud.Server, autoscalingPools []config.WorkerNodePool, masterClusterIP string, k3sToken string) error {
	spec, ok := doc["spec"].(map[string]interface{})
	if !ok {
		return nil
	}

	template, ok := spec["template"].(map[string]interface{})
	if !ok {
		return nil
	}

	podSpec, ok := template["spec"].(map[string]interface{})
	if !ok {
		return nil
	}

	// Add tolerations for running on master nodes
	tolerations := []map[string]interface{}{
		{
			"key":      "CriticalAddonsOnly",
			"operator": "Exists",
		},
		{
			"key":      "node-role.kubernetes.io/control-plane",
			"operator": "Exists",
			"effect":   "NoSchedule",
		},
		{
			"key":      "node-role.kubernetes.io/master",
			"operator": "Exists",
			"effect":   "NoSchedule",
		},
	}
	podSpec["tolerations"] = tolerations

	// Add node affinity to prefer running on master nodes
	// This implements the "master A -> autoscaler A -> pool A" locality pattern
	// The autoscaler will preferentially run on master nodes, creating regional affinity
	affinity := map[string]interface{}{
		"nodeAffinity": map[string]interface{}{
			"preferredDuringSchedulingIgnoredDuringExecution": []map[string]interface{}{
				{
					"weight": 100,
					"preference": map[string]interface{}{
						"matchExpressions": []map[string]interface{}{
							{
								"key":      "node-role.kubernetes.io/control-plane",
								"operator": "Exists",
							},
						},
					},
				},
				{
					"weight": 100,
					"preference": map[string]interface{}{
						"matchExpressions": []map[string]interface{}{
							{
								"key":      "node-role.kubernetes.io/master",
								"operator": "Exists",
							},
						},
					},
				},
			},
		},
	}
	podSpec["affinity"] = affinity

	// Patch containers
	containers, ok := podSpec["containers"].([]interface{})
	if !ok || len(containers) == 0 {
		return nil
	}

	for i, cont := range containers {
		container, ok := cont.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := container["name"].(string)
		if name == "cluster-autoscaler" {
			if err := c.patchAutoscalerContainer(container, firstMaster, masters, autoscalingPools, masterClusterIP, k3sToken); err != nil {
				return err
			}
			containers[i] = container
		}
	}
	podSpec["containers"] = containers

	return nil
}

// patchAutoscalerContainer patches the cluster-autoscaler container
func (c *ClusterAutoscalerInstaller) patchAutoscalerContainer(container map[string]interface{}, firstMaster *hcloud.Server, masters []*hcloud.Server, autoscalingPools []config.WorkerNodePool, masterClusterIP string, k3sToken string) error {
	// Update image
	container["image"] = fmt.Sprintf("registry.k8s.io/autoscaling/cluster-autoscaler:%s", c.Config.Addons.ClusterAutoscaler.ContainerImageTag)

	// Build command with node pool args
	command := []string{
		"./cluster-autoscaler",
		"--cloud-provider=hetzner",
		"--enforce-node-group-min-size",
	}

	// Add node pool arguments
	// For multi-location autoscaling pools, create one node group per location
	// This enables the autoscaler to manage nodes independently in each region
	for _, pool := range autoscalingPools {
		// Create a node group for each location in the pool
		for i, location := range pool.Locations {
			poolName := pool.BuildNodePoolName(c.Config.ClusterName)

			// For multi-location pools, append location suffix to distinguish node groups
			// Example: "my-cluster-workers" becomes "my-cluster-workers-fsn1", "my-cluster-workers-hel1"
			if len(pool.Locations) > 1 {
				// Use short location code (e.g., "fsn1" -> "fsn1")
				poolName = fmt.Sprintf("%s-%s", poolName, location)
			}

			// Distribute min/max instances across locations for even distribution
			// This ensures each location gets approximately equal capacity
			minInstances := pool.Autoscaling.MinInstances / len(pool.Locations)
			maxInstances := pool.Autoscaling.MaxInstances / len(pool.Locations)

			// For the first location, add any remainder from integer division
			// This ensures we don't lose capacity due to rounding
			if i == 0 {
				minInstances += pool.Autoscaling.MinInstances % len(pool.Locations)
				maxInstances += pool.Autoscaling.MaxInstances % len(pool.Locations)
			}

			nodePoolArg := fmt.Sprintf("--nodes=%d:%d:%s:%s:%s",
				minInstances,
				maxInstances,
				strings.ToUpper(pool.InstanceType),
				strings.ToUpper(location),
				poolName,
			)
			command = append(command, nodePoolArg)
		}
	}

	// Add autoscaler config args
	autoscalerCfg := c.Config.Addons.ClusterAutoscaler
	command = append(command,
		fmt.Sprintf("--scan-interval=%s", autoscalerCfg.ScanInterval),
		fmt.Sprintf("--scale-down-delay-after-add=%s", autoscalerCfg.ScaleDownDelayAfterAdd),
		fmt.Sprintf("--scale-down-delay-after-delete=%s", autoscalerCfg.ScaleDownDelayAfterDelete),
		fmt.Sprintf("--scale-down-delay-after-failure=%s", autoscalerCfg.ScaleDownDelayAfterFailure),
		fmt.Sprintf("--max-node-provision-time=%s", autoscalerCfg.MaxNodeProvisionTime),
	)

	// Add custom args if any
	command = append(command, c.Config.ClusterAutoscalerArgs...)

	container["command"] = command

	// Build environment variables
	env, err := c.buildEnvironmentVariables(firstMaster, masters, autoscalingPools, masterClusterIP, k3sToken)
	if err != nil {
		return err
	}
	container["env"] = env

	return nil
}

// buildEnvironmentVariables builds the environment variables for the cluster autoscaler
func (c *ClusterAutoscalerInstaller) buildEnvironmentVariables(firstMaster *hcloud.Server, masters []*hcloud.Server, autoscalingPools []config.WorkerNodePool, masterClusterIP string, k3sToken string) ([]map[string]interface{}, error) {
	// Build cluster config JSON
	clusterConfig, err := c.buildClusterConfig(firstMaster, masters, autoscalingPools, masterClusterIP, k3sToken)
	if err != nil {
		return nil, err
	}

	// Encode to base64
	clusterConfigBase64 := base64.StdEncoding.EncodeToString([]byte(clusterConfig))

	// Determine network name using shared utility function
	networkName := util.ResolveNetworkName(c.Config)

	// Determine if public IPs should be enabled
	// When NAT gateway is enabled, nodes should not have public IPs
	enablePublicIPv4 := false
	enablePublicIPv6 := false

	// Check if public network IPv4 is configured and enabled
	if c.Config.Networking.PublicNetwork.IPv4 != nil {
		enablePublicIPv4 = c.Config.Networking.PublicNetwork.IPv4.Enabled
	}

	// Check if public network IPv6 is configured and enabled
	if c.Config.Networking.PublicNetwork.IPv6 != nil {
		enablePublicIPv6 = c.Config.Networking.PublicNetwork.IPv6.Enabled
	}

	// If NAT gateway is enabled, force public IPs to false
	if c.Config.Networking.PrivateNetwork.NATGateway != nil && c.Config.Networking.PrivateNetwork.NATGateway.Enabled {
		enablePublicIPv4 = false
		enablePublicIPv6 = false
	}

	env := []map[string]interface{}{
		{
			"name": "HCLOUD_TOKEN",
			"valueFrom": map[string]interface{}{
				"secretKeyRef": map[string]interface{}{
					"name": "hcloud",
					"key":  "token",
				},
			},
		},
		{
			"name":  "HCLOUD_CLUSTER_CONFIG",
			"value": clusterConfigBase64,
		},
		{
			"name":  "HCLOUD_FIREWALL",
			"value": c.Config.ClusterName,
		},
		{
			"name":  "HCLOUD_SSH_KEY",
			"value": c.Config.ClusterName,
		},
		{
			"name":  "HCLOUD_NETWORK",
			"value": networkName,
		},
		{
			"name":  "HCLOUD_PUBLIC_IPV4",
			"value": fmt.Sprintf("%t", enablePublicIPv4),
		},
		{
			"name":  "HCLOUD_PUBLIC_IPV6",
			"value": fmt.Sprintf("%t", enablePublicIPv6),
		},
	}

	return env, nil
}

// buildClusterConfig builds the cluster configuration JSON
func (c *ClusterAutoscalerInstaller) buildClusterConfig(firstMaster *hcloud.Server, masters []*hcloud.Server, autoscalingPools []config.WorkerNodePool, masterClusterIP string, k3sToken string) (string, error) {
	image := c.Config.Image
	if c.Config.AutoscalingImage != "" {
		image = c.Config.AutoscalingImage
	}

	nodeConfigs := make(map[string]interface{})
	for _, pool := range autoscalingPools {
		// Create a node config for each location in the pool
		// This ensures the autoscaler provider creates servers with the correct location suffix in their names
		for _, location := range pool.Locations {
			poolName := pool.BuildNodePoolName(c.Config.ClusterName)

			// For multi-location pools, append location suffix to distinguish node groups
			// This matches the naming pattern used in patchAutoscalerContainer
			if len(pool.Locations) > 1 {
				poolName = fmt.Sprintf("%s-%s", poolName, location)
			}

			nodeConfig, err := c.buildNodeConfig(pool, location, firstMaster, masters, masterClusterIP, k3sToken)
			if err != nil {
				return "", err
			}
			nodeConfigs[poolName] = nodeConfig
		}
	}

	config := map[string]interface{}{
		"imagesForArch": map[string]string{
			"arm64": image,
			"amd64": image,
		},
		"nodeConfigs": nodeConfigs,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cluster config: %w", err)
	}

	return string(configJSON), nil
}

// buildNodeConfig builds the configuration for a node pool
func (c *ClusterAutoscalerInstaller) buildNodeConfig(pool config.WorkerNodePool, location string, firstMaster *hcloud.Server, masters []*hcloud.Server, masterClusterIP string, k3sToken string) (map[string]interface{}, error) {
	// Generate cloud-init for the pool
	// The cloud-init script contains the worker install script which sets node labels and taints
	// via kubelet flags (--node-label and --node-taint) when the node joins the cluster
	cloudInitData, err := c.generateCloudInitForPool(pool, firstMaster, masters, masterClusterIP, k3sToken)
	if err != nil {
		return nil, err
	}

	// Build Hetzner Cloud server labels to match static worker nodes
	// These are Hetzner Cloud metadata labels, NOT Kubernetes labels
	// This ensures autoscaled nodes have the same labeling scheme as static nodes
	poolName := pool.Name
	if poolName == nil {
		defaultName := "default"
		poolName = &defaultName
	}

	// Autoscaler only supports default labels, not custom Hetzner labels
	// Custom Hetzner labels are only supported for static worker pools
	serverLabels := map[string]string{
		"cluster":  c.Config.ClusterName,
		"role":     "worker",
		"pool":     *poolName,
		"location": location,
		"managed":  "kuberaptor",
	}

	// Note: We do NOT pass Kubernetes labels and taints to the autoscaler provider
	// The Hetzner autoscaler provider automatically sets ONLY the 'hcloud/node-group' label
	// All other custom labels and taints MUST be set via kubelet flags (--node-label, --node-taint)
	// in the cloud-init script when the node joins the cluster (see generateWorkerInstallScript)
	// This is the correct approach as the cluster autoscaler is not responsible for setting these
	return map[string]interface{}{
		"cloudInit":    cloudInitData,
		"serverLabels": serverLabels,
	}, nil
}

// generateCloudInitForPool generates cloud-init data for a worker pool
func (c *ClusterAutoscalerInstaller) generateCloudInitForPool(pool config.WorkerNodePool, firstMaster *hcloud.Server, masters []*hcloud.Server, masterClusterIP string, k3sToken string) (string, error) {
	// Generate worker install script
	workerScript, err := c.generateWorkerInstallScript(masterClusterIP, pool, k3sToken)
	if err != nil {
		return "", fmt.Errorf("failed to generate worker install script: %w", err)
	}

	// Combine all packages
	allPackages := append([]string{}, c.Config.AdditionalPackages...)
	allPackages = append(allPackages, pool.AdditionalPackages...)

	// Combine pre-k3s commands
	initCommands := []string{}
	initCommands = append(initCommands, c.Config.AdditionalPreK3sCommands...)
	initCommands = append(initCommands, pool.AdditionalPreK3sCommands...)
	// Add the worker install script as the main init command
	initCommands = append(initCommands, workerScript)
	// Add post-k3s commands
	initCommands = append(initCommands, c.Config.AdditionalPostK3sCommands...)
	initCommands = append(initCommands, pool.AdditionalPostK3sCommands...)

	// Generate cloud-init with init commands
	generator := cloudinit.NewGenerator(&cloudinit.Config{
		SSHPort:            c.Config.Networking.SSH.Port,
		Packages:           allPackages,
		InitCommands:       initCommands,
		ClusterCIDR:        c.Config.Networking.ClusterCIDR,
		ServiceCIDR:        c.Config.Networking.ServiceCIDR,
		AllowedNetworksSSH: c.Config.Networking.AllowedNetworks.SSH,
		AllowedNetworksAPI: c.Config.Networking.AllowedNetworks.API,
	})

	cloudInit, err := generator.Generate()
	if err != nil {
		return "", fmt.Errorf("failed to generate cloud-init: %w", err)
	}

	return cloudInit, nil
}

// generateWorkerInstallScript generates the worker install script from template
func (c *ClusterAutoscalerInstaller) generateWorkerInstallScript(masterIP string, pool config.WorkerNodePool, k3sToken string) (string, error) {
	// Build node labels with escaping using Kubernetes labels
	nodeLabels := []string{}
	kubernetesLabels := pool.KubernetesLabels()
	for _, label := range kubernetesLabels {
		// Escape special characters to prevent shell injection
		key := util.EscapeShellArg(label.Key)
		value := util.EscapeShellArg(label.Value)
		nodeLabels = append(nodeLabels, fmt.Sprintf("%s=%s", key, value))
	}

	// Build node taints with escaping using Kubernetes taints
	nodeTaints := []string{}
	kubernetesTaints := pool.KubernetesTaints()
	for _, taint := range kubernetesTaints {
		// Escape special characters to prevent shell injection
		key := util.EscapeShellArg(taint.Key)
		value := util.EscapeShellArg(taint.Value)
		effect := util.EscapeShellArg(taint.Effect)
		nodeTaints = append(nodeTaints, fmt.Sprintf("%s=%s:%s", key, value, effect))
	}

	// Build labels and taints string
	labelsAndTaintsStr := c.buildLabelsAndTaintsString(nodeLabels, nodeTaints)

	// Build kubelet args
	kubeletArgs := c.buildKubeletArgsString()

	// Prepare template data
	data := map[string]interface{}{
		"PrivateNetworkEnabled": c.Config.Networking.PrivateNetwork.Enabled,
		"PrivateNetworkSubnet":  c.Config.Networking.PrivateNetwork.Subnet,
		"K3sToken":              k3sToken,
		"K3sVersion":            c.Config.K3sVersion,
		"MasterIP":              masterIP,
		"KubeletArgs":           kubeletArgs,
		"LabelsAndTaints":       labelsAndTaintsStr,
	}

	// Parse and execute template
	tmpl, err := template.New("worker-install").Parse(workerInstallScriptTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse worker install script template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute worker install script template: %w", err)
	}

	return buf.String(), nil
}

// buildLabelsAndTaintsString builds the labels and taints string for k3s
func (c *ClusterAutoscalerInstaller) buildLabelsAndTaintsString(labels []string, taints []string) string {
	parts := []string{}

	if len(labels) > 0 {
		parts = append(parts, fmt.Sprintf("--node-label=%s", strings.Join(labels, ",")))
	}

	if len(taints) > 0 {
		parts = append(parts, fmt.Sprintf("--node-taint=%s", strings.Join(taints, ",")))
	}

	return strings.Join(parts, " ")
}

// buildKubeletArgsString builds the kubelet args string
func (c *ClusterAutoscalerInstaller) buildKubeletArgsString() string {
	allArgs := c.Config.AllKubeletArgs()
	if len(allArgs) == 0 {
		return ""
	}

	var args []string
	for _, arg := range allArgs {
		args = append(args, fmt.Sprintf("--kubelet-arg=%s", arg))
	}

	return strings.Join(args, " ")
}

// patchClusterRole patches the ClusterRole to add volumeattachments permission
func (c *ClusterAutoscalerInstaller) patchClusterRole(doc map[string]interface{}) {
	rules, ok := doc["rules"].([]interface{})
	if !ok {
		return
	}

	// Find the storage.k8s.io rule and add volumeattachments if not present
	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}

		apiGroups, ok := ruleMap["apiGroups"].([]interface{})
		if !ok {
			continue
		}

		hasStorageAPI := false
		for _, group := range apiGroups {
			if groupStr, ok := group.(string); ok && groupStr == "storage.k8s.io" {
				hasStorageAPI = true
				break
			}
		}

		if !hasStorageAPI {
			continue
		}

		resources, ok := ruleMap["resources"].([]interface{})
		if !ok {
			continue
		}

		// Check if volumeattachments is already present
		hasVolumeAttachments := false
		for _, res := range resources {
			if resStr, ok := res.(string); ok && resStr == "volumeattachments" {
				hasVolumeAttachments = true
				break
			}
		}

		if !hasVolumeAttachments {
			resources = append(resources, "volumeattachments")
			ruleMap["resources"] = resources
		}
	}
}
