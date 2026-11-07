// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package addons

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/internal/config"
	"gopkg.in/yaml.v3"
)

// sampleAutoscalerManifest mirrors the upstream Hetzner example manifest structure
const sampleAutoscalerManifest = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: cluster-autoscaler
  namespace: kube-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cluster-autoscaler
  namespace: kube-system
spec:
  template:
    spec:
      containers:
        - image: registry.k8s.io/autoscaling/cluster-autoscaler:latest
          name: cluster-autoscaler
          command:
            - ./cluster-autoscaler
          volumeMounts:
            - name: ssl-certs
              mountPath: /etc/ssl/certs/ca-certificates.crt
              readOnly: true
      volumes:
        - name: ssl-certs
          hostPath:
            path: "/etc/ssl/certs/ca-certificates.crt"
`

func newFeatureTestConfig() *config.Main {
	return &config.Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.37.2+k3s1",
		Image:       "debian-13",
		Networking: config.Networking{
			PrivateNetwork: config.PrivateNetwork{Enabled: true, Subnet: "10.0.0.0/16"},
			SSH:            config.SSH{Port: 22},
		},
		Addons: config.Addons{
			ClusterAutoscaler: &config.ClusterAutoscaler{
				Enabled:                    true,
				ContainerImageTag:          "v1.37.0",
				ScanInterval:               "10s",
				ScaleDownDelayAfterAdd:     "10m",
				ScaleDownDelayAfterDelete:  "10s",
				ScaleDownDelayAfterFailure: "3m",
				MaxNodeProvisionTime:       "15m",
			},
		},
	}
}

func newFeatureTestPool(autoscaling *config.Autoscaling) config.WorkerNodePool {
	return config.WorkerNodePool{
		NodePool: config.NodePool{
			Name:                       stringPtr("workers"),
			InstanceType:               "cpx32",
			IncludeClusterNameAsPrefix: true,
			Autoscaling:                autoscaling,
		},
		Locations: []string{"fsn1"},
	}
}

// TestNodeConfigProviderSettingsUnset verifies that a pool without the optional
// provider settings produces exactly the same node config keys as before.
func TestNodeConfigProviderSettingsUnset(t *testing.T) {
	installer := NewClusterAutoscalerInstaller(newFeatureTestConfig(), nil)
	pool := newFeatureTestPool(&config.Autoscaling{Enabled: true, MinInstances: 0, MaxInstances: 3})
	master := &hcloud.Server{Name: "test-master-1"}

	nodeConfig, err := installer.buildNodeConfig(pool, "fsn1", master, []*hcloud.Server{master}, "10.0.0.1", "token")
	if err != nil {
		t.Fatalf("Failed to build node config: %v", err)
	}

	var keys []string
	for key := range nodeConfig {
		keys = append(keys, key)
	}
	if len(keys) != 2 || nodeConfig["cloudInit"] == nil || nodeConfig["serverLabels"] == nil {
		t.Errorf("node config keys = %v, expected only cloudInit and serverLabels", keys)
	}

	clusterConfig, err := installer.buildClusterConfig(master, []*hcloud.Server{master}, []config.WorkerNodePool{pool}, "10.0.0.1", "token")
	if err != nil {
		t.Fatalf("Failed to build cluster config: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(clusterConfig), &parsed); err != nil {
		t.Fatalf("Failed to parse cluster config: %v", err)
	}
	if _, ok := parsed["defaultSubnetIPRange"]; ok {
		t.Error("cluster config should not contain defaultSubnetIPRange when it is not configured")
	}
}

// TestNodeConfigProviderSettings verifies the optional per-pool provider settings
func TestNodeConfigProviderSettings(t *testing.T) {
	cfg := newFeatureTestConfig()
	cfg.Addons.ClusterAutoscaler.DefaultSubnetIPRange = "10.0.0.0/16"
	installer := NewClusterAutoscalerInstaller(cfg, nil)
	pool := newFeatureTestPool(&config.Autoscaling{
		Enabled:        true,
		MinInstances:   0,
		MaxInstances:   3,
		Image:          "ubuntu-24.04",
		SubnetIPRange:  "10.0.0.0/16",
		Firewalls:      []string{"extra-firewall", "12345"},
		PlacementGroup: "test-cluster-workers",
	})
	master := &hcloud.Server{Name: "test-master-1"}

	clusterConfig, err := installer.buildClusterConfig(master, []*hcloud.Server{master}, []config.WorkerNodePool{pool}, "10.0.0.1", "token")
	if err != nil {
		t.Fatalf("Failed to build cluster config: %v", err)
	}

	// Decode the way the upstream provider does (case-insensitive field match)
	var parsed struct {
		ImagesForArch        struct{ Arm64, Amd64 string }
		DefaultSubnetIPRange string
		NodeConfigs          map[string]struct {
			PlacementGroup string
			ImagesForArch  *struct{ Arm64, Amd64 string }
			SubnetIPRange  string
			Firewalls      []string
		}
	}
	if err := json.Unmarshal([]byte(clusterConfig), &parsed); err != nil {
		t.Fatalf("Failed to parse cluster config: %v", err)
	}

	if parsed.ImagesForArch.Amd64 != "debian-13" {
		t.Errorf("global image = %q, expected %q", parsed.ImagesForArch.Amd64, "debian-13")
	}
	if parsed.DefaultSubnetIPRange != "10.0.0.0/16" {
		t.Errorf("defaultSubnetIPRange = %q, expected %q", parsed.DefaultSubnetIPRange, "10.0.0.0/16")
	}

	nodeConfig, ok := parsed.NodeConfigs["test-cluster-workers"]
	if !ok {
		t.Fatalf("node config for test-cluster-workers missing: %v", parsed.NodeConfigs)
	}
	if nodeConfig.ImagesForArch == nil || nodeConfig.ImagesForArch.Arm64 != "ubuntu-24.04" || nodeConfig.ImagesForArch.Amd64 != "ubuntu-24.04" {
		t.Errorf("pool imagesForArch = %+v, expected ubuntu-24.04 for both architectures", nodeConfig.ImagesForArch)
	}
	if nodeConfig.SubnetIPRange != "10.0.0.0/16" {
		t.Errorf("subnetIPRange = %q, expected %q", nodeConfig.SubnetIPRange, "10.0.0.0/16")
	}
	if !reflect.DeepEqual(nodeConfig.Firewalls, []string{"extra-firewall", "12345"}) {
		t.Errorf("firewalls = %v, expected [extra-firewall 12345]", nodeConfig.Firewalls)
	}
	if nodeConfig.PlacementGroup != "test-cluster-workers" {
		t.Errorf("placementGroup = %q, expected %q", nodeConfig.PlacementGroup, "test-cluster-workers")
	}
}

// findDeploymentAndSecret parses a patched manifest
func findDeploymentAndSecret(t *testing.T, manifest string) (firstKind string, deployment, secret map[string]interface{}) {
	t.Helper()
	for i, resource := range strings.Split(manifest, "---\n") {
		var doc map[string]interface{}
		if err := yaml.Unmarshal([]byte(resource), &doc); err != nil {
			t.Fatalf("Failed to parse resource %d: %v", i, err)
		}
		kind, _ := doc["kind"].(string)
		if i == 0 {
			firstKind = kind
		}
		switch kind {
		case "Deployment":
			deployment = doc
		case "Secret":
			secret = doc
		}
	}
	if deployment == nil {
		t.Fatal("Deployment not found in patched manifest")
	}
	return firstKind, deployment, secret
}

func deploymentPodSpec(deployment map[string]interface{}) (map[string]interface{}, map[string]interface{}) {
	podSpec := deployment["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})
	container := podSpec["containers"].([]interface{})[0].(map[string]interface{})
	return podSpec, container
}

func envNames(container map[string]interface{}) map[string]string {
	result := map[string]string{}
	for _, item := range container["env"].([]interface{}) {
		env := item.(map[string]interface{})
		value, _ := env["value"].(string)
		result[env["name"].(string)] = value
	}
	return result
}

func hasNamedItem(items interface{}, name string) bool {
	list, _ := items.([]interface{})
	for _, item := range list {
		if m, ok := item.(map[string]interface{}); ok && m["name"] == name {
			return true
		}
	}
	return false
}

// TestPatchManifestClusterConfigEnv verifies the default delivery through the
// HCLOUD_CLUSTER_CONFIG environment variable is unchanged.
func TestPatchManifestClusterConfigEnv(t *testing.T) {
	installer := NewClusterAutoscalerInstaller(newFeatureTestConfig(), nil)
	pool := newFeatureTestPool(&config.Autoscaling{Enabled: true, MinInstances: 0, MaxInstances: 3})
	master := &hcloud.Server{Name: "test-master-1"}

	manifest, err := installer.patchManifest(sampleAutoscalerManifest, master, []*hcloud.Server{master}, []config.WorkerNodePool{pool}, "10.0.0.1", "token")
	if err != nil {
		t.Fatalf("Failed to patch manifest: %v", err)
	}

	_, deployment, secret := findDeploymentAndSecret(t, manifest)
	if secret != nil {
		t.Error("manifest should not contain a Secret by default")
	}
	podSpec, container := deploymentPodSpec(deployment)
	env := envNames(container)
	if env["HCLOUD_CLUSTER_CONFIG"] == "" {
		t.Error("HCLOUD_CLUSTER_CONFIG should be set by default")
	}
	if _, ok := env["HCLOUD_CLUSTER_CONFIG_FILE"]; ok {
		t.Error("HCLOUD_CLUSTER_CONFIG_FILE should not be set by default")
	}
	if hasNamedItem(podSpec["volumes"], clusterConfigVolumeName) || hasNamedItem(container["volumeMounts"], clusterConfigVolumeName) {
		t.Error("cluster config volume should not be mounted by default")
	}
	if !hasNamedItem(podSpec["volumes"], "ssl-certs") {
		t.Error("upstream ssl-certs volume should be kept")
	}
}

// TestPatchManifestClusterConfigSecret verifies delivery through a mounted Secret file
func TestPatchManifestClusterConfigSecret(t *testing.T) {
	cfg := newFeatureTestConfig()
	cfg.Addons.ClusterAutoscaler.ClusterConfigFromSecret = true
	installer := NewClusterAutoscalerInstaller(cfg, nil)
	pool := newFeatureTestPool(&config.Autoscaling{Enabled: true, MinInstances: 0, MaxInstances: 3})
	master := &hcloud.Server{Name: "test-master-1"}

	manifest, err := installer.patchManifest(sampleAutoscalerManifest, master, []*hcloud.Server{master}, []config.WorkerNodePool{pool}, "10.0.0.1", "token")
	if err != nil {
		t.Fatalf("Failed to patch manifest: %v", err)
	}
	assertSecretDelivery(t, manifest)
}

// TestPatchManifestLargeClusterConfigUsesSecret verifies that a configuration too
// large for an environment variable is delivered through a Secret file.
func TestPatchManifestLargeClusterConfigUsesSecret(t *testing.T) {
	cfg := newFeatureTestConfig()
	// Incompressible commands make the gzip-compressed cloud-init large
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 2000; i++ {
		cfg.AdditionalPostK3sCommands = append(cfg.AdditionalPostK3sCommands, fmt.Sprintf("echo %016x%016x%016x%016x", rng.Uint64(), rng.Uint64(), rng.Uint64(), rng.Uint64()))
	}
	installer := NewClusterAutoscalerInstaller(cfg, nil)
	pool := newFeatureTestPool(&config.Autoscaling{Enabled: true, MinInstances: 0, MaxInstances: 3})
	master := &hcloud.Server{Name: "test-master-1"}

	manifest, err := installer.patchManifest(sampleAutoscalerManifest, master, []*hcloud.Server{master}, []config.WorkerNodePool{pool}, "10.0.0.1", "token")
	if err != nil {
		t.Fatalf("Failed to patch manifest: %v", err)
	}
	assertSecretDelivery(t, manifest)
}

func assertSecretDelivery(t *testing.T, manifest string) {
	t.Helper()
	firstKind, deployment, secret := findDeploymentAndSecret(t, manifest)
	if secret == nil {
		t.Fatal("manifest should contain the cluster config Secret")
	}
	if firstKind != "Secret" {
		t.Errorf("first resource kind = %q, expected the Secret to be applied first", firstKind)
	}
	metadata := secret["metadata"].(map[string]interface{})
	if metadata["name"] != clusterConfigSecretName || metadata["namespace"] != "kube-system" {
		t.Errorf("Secret metadata = %v", metadata)
	}
	data := secret["stringData"].(map[string]interface{})[clusterConfigFileName].(string)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		t.Fatalf("Secret cluster config is not valid JSON: %v", err)
	}
	if _, ok := parsed["nodeConfigs"]; !ok {
		t.Error("Secret cluster config should contain nodeConfigs")
	}

	podSpec, container := deploymentPodSpec(deployment)
	env := envNames(container)
	if _, ok := env["HCLOUD_CLUSTER_CONFIG"]; ok {
		t.Error("HCLOUD_CLUSTER_CONFIG should not be set when the Secret file is used")
	}
	if env["HCLOUD_CLUSTER_CONFIG_FILE"] != clusterConfigMountPath+"/"+clusterConfigFileName {
		t.Errorf("HCLOUD_CLUSTER_CONFIG_FILE = %q", env["HCLOUD_CLUSTER_CONFIG_FILE"])
	}
	if !hasNamedItem(podSpec["volumes"], clusterConfigVolumeName) {
		t.Error("cluster config Secret volume missing")
	}
	if !hasNamedItem(container["volumeMounts"], clusterConfigVolumeName) {
		t.Error("cluster config Secret volume mount missing")
	}
	if !hasNamedItem(podSpec["volumes"], "ssl-certs") || !hasNamedItem(container["volumeMounts"], "ssl-certs") {
		t.Error("upstream ssl-certs volume should be kept")
	}
}
