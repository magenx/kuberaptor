// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package addons

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/internal/util"
	"gopkg.in/yaml.v3"
)

// KuredInstaller installs Kured (Kubernetes Reboot Daemon)
type KuredInstaller struct {
	Config        *config.Main
	SSHClient     *util.SSH
	KubectlClient *util.KubectlClient
	ctx           context.Context
}

// NewKuredInstaller creates a new Kured installer
func NewKuredInstaller(cfg *config.Main, sshClient *util.SSH) *KuredInstaller {
	return &KuredInstaller{
		Config:        cfg,
		SSHClient:     sshClient,
		KubectlClient: util.NewKubectlClient(cfg.KubeconfigPath),
		ctx:           context.Background(),
	}
}

// Install installs Kured using local kubectl
func (k *KuredInstaller) Install(firstMaster *hcloud.Server, masterIP string) error {
	// Check if Kured is already installed
	if k.KubectlClient.ResourceExists("daemonset", "kured", "kube-system") {
		util.LogInfo("Kured already installed, skipping installation", "addons")
		return nil
	}

	// Fetch and patch the manifest
	manifest, err := k.generateManifest()
	if err != nil {
		return fmt.Errorf("failed to generate Kured manifest: %w", err)
	}

	// Apply using local kubectl
	if err := k.KubectlClient.ApplyManifest(manifest); err != nil {
		return fmt.Errorf("failed to apply Kured manifest: %w", err)
	}

	util.LogSuccess("Kured installed", "addons")
	return nil
}

// generateManifest fetches and patches the Kured manifest
func (k *KuredInstaller) generateManifest() (string, error) {
	manifestURL := k.Config.Addons.Kured.ManifestURL

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Get(manifestURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Kured manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch Kured manifest: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Kured manifest: %w", err)
	}

	return k.patchManifest(string(body))
}

// patchManifest parses and patches each resource in the manifest
func (k *KuredInstaller) patchManifest(manifestStr string) (string, error) {
	// Normalize line endings and split on YAML document separators
	manifestStr = strings.ReplaceAll(manifestStr, "\r\n", "\n")
	resources := strings.Split(manifestStr, "\n---")

	var patchedResources []string
	for _, resource := range resources {
		resource = strings.TrimSpace(resource)
		if resource == "" {
			continue
		}

		var doc map[string]interface{}
		if err := yaml.Unmarshal([]byte(resource), &doc); err != nil {
			// If parsing fails, keep the original resource
			patchedResources = append(patchedResources, resource)
			continue
		}

		kind, _ := doc["kind"].(string)
		if kind == "DaemonSet" {
			k.patchDaemonSet(doc)
		}

		patchedBytes, err := yaml.Marshal(doc)
		if err != nil {
			return "", fmt.Errorf("failed to marshal patched Kured resource: %w", err)
		}
		patchedResources = append(patchedResources, string(patchedBytes))
	}

	return strings.Join(patchedResources, "\n---\n"), nil
}

// patchDaemonSet patches the Kured DaemonSet to add kured_options and master node tolerations
func (k *KuredInstaller) patchDaemonSet(doc map[string]interface{}) {
	spec, ok := doc["spec"].(map[string]interface{})
	if !ok {
		return
	}

	template, ok := spec["template"].(map[string]interface{})
	if !ok {
		return
	}

	podSpec, ok := template["spec"].(map[string]interface{})
	if !ok {
		return
	}

	// Add tolerations to allow Kured to run on master/control-plane nodes
	k.patchTolerations(podSpec)

	// Patch container command to append kured_options
	containers, ok := podSpec["containers"].([]interface{})
	if !ok || len(containers) == 0 {
		return
	}

	for i, cont := range containers {
		container, ok := cont.(map[string]interface{})
		if !ok {
			continue
		}
		k.patchContainerCommand(container)
		containers[i] = container
	}
	podSpec["containers"] = containers
}

// patchTolerations adds tolerations for master and control-plane nodes to the pod spec
func (k *KuredInstaller) patchTolerations(podSpec map[string]interface{}) {
	masterTolerations := []interface{}{
		map[string]interface{}{
			"key":      "CriticalAddonsOnly",
			"operator": "Exists",
		},
		map[string]interface{}{
			"key":      "node-role.kubernetes.io/control-plane",
			"operator": "Exists",
			"effect":   "NoSchedule",
		},
		map[string]interface{}{
			"key":      "node-role.kubernetes.io/master",
			"operator": "Exists",
			"effect":   "NoSchedule",
		},
	}

	// Merge with any existing tolerations
	if existing, ok := podSpec["tolerations"].([]interface{}); ok {
		podSpec["tolerations"] = append(existing, masterTolerations...)
	} else {
		podSpec["tolerations"] = masterTolerations
	}
}

// patchContainerCommand appends kured_options to the container command
func (k *KuredInstaller) patchContainerCommand(container map[string]interface{}) {
	if len(k.Config.Addons.Kured.KuredOptions) == 0 {
		return
	}

	cmd, ok := container["command"].([]interface{})
	if !ok {
		return
	}

	for _, opt := range k.Config.Addons.Kured.KuredOptions {
		cmd = append(cmd, opt)
	}
	container["command"] = cmd
}
