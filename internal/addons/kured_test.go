// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package addons

import (
	"strings"
	"testing"

	"github.com/magenx/kuberaptor/internal/config"
	"gopkg.in/yaml.v3"
)

// sampleKuredDaemonSet is a minimal Kured DaemonSet manifest for testing
const sampleKuredDaemonSet = `apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: kured
  namespace: kube-system
spec:
  selector:
    matchLabels:
      name: kured
  template:
    metadata:
      labels:
        name: kured
    spec:
      containers:
        - name: kured
          image: ghcr.io/kubereboot/kured:1.21.0
          command:
            - /usr/bin/kured
            - --reboot-sentinel=/sentinel/reboot-required
`

// TestKuredPatchManifest_AppendOptions verifies that kured_options are appended to
// the container command section in the DaemonSet manifest
func TestKuredPatchManifest_AppendOptions(t *testing.T) {
	cfg := &config.Main{
		KubeconfigPath: "/tmp/kubeconfig",
		Addons: config.Addons{
			Kured: &config.Kured{
				Enabled:     true,
				ManifestURL: "https://example.com/kured.yaml",
				KuredOptions: []string{
					"--reboot-command=/usr/bin/systemctl reboot",
					"--period=60m",
					"--reboot-days=mon,tue,wed,thu,fri,sat,sun",
					"--start-time=1am",
					"--end-time=8am",
					"--time-zone=Europe/Berlin",
				},
			},
		},
	}

	installer := NewKuredInstaller(cfg, nil)

	patched, err := installer.patchManifest(sampleKuredDaemonSet)
	if err != nil {
		t.Fatalf("patchManifest returned error: %v", err)
	}

	// Each kured_option must appear in the patched manifest
	for _, opt := range cfg.Addons.Kured.KuredOptions {
		if !strings.Contains(patched, opt) {
			t.Errorf("patched manifest should contain kured option %q", opt)
		}
	}

	// The original sentinel option must still be present
	if !strings.Contains(patched, "--reboot-sentinel=/sentinel/reboot-required") {
		t.Error("patched manifest should still contain the original --reboot-sentinel option")
	}
}

// TestKuredPatchManifest_NoOptions verifies that when KuredOptions is empty,
// the manifest is still valid and the original command is unchanged
func TestKuredPatchManifest_NoOptions(t *testing.T) {
	cfg := &config.Main{
		KubeconfigPath: "/tmp/kubeconfig",
		Addons: config.Addons{
			Kured: &config.Kured{
				Enabled:      true,
				ManifestURL:  "https://example.com/kured.yaml",
				KuredOptions: nil,
			},
		},
	}

	installer := NewKuredInstaller(cfg, nil)

	patched, err := installer.patchManifest(sampleKuredDaemonSet)
	if err != nil {
		t.Fatalf("patchManifest returned error: %v", err)
	}

	// Original command should still be present
	if !strings.Contains(patched, "--reboot-sentinel=/sentinel/reboot-required") {
		t.Error("patched manifest should contain the original --reboot-sentinel option")
	}
}

// TestKuredPatchManifest_MasterTolerations verifies that master node tolerations
// are injected into the DaemonSet pod spec
func TestKuredPatchManifest_MasterTolerations(t *testing.T) {
	cfg := &config.Main{
		KubeconfigPath: "/tmp/kubeconfig",
		Addons: config.Addons{
			Kured: &config.Kured{
				Enabled:     true,
				ManifestURL: "https://example.com/kured.yaml",
			},
		},
	}

	installer := NewKuredInstaller(cfg, nil)

	patched, err := installer.patchManifest(sampleKuredDaemonSet)
	if err != nil {
		t.Fatalf("patchManifest returned error: %v", err)
	}

	// Parse the patched manifest to inspect tolerations
	var doc map[string]interface{}
	if err := yaml.Unmarshal([]byte(patched), &doc); err != nil {
		t.Fatalf("failed to parse patched manifest YAML: %v", err)
	}

	spec := doc["spec"].(map[string]interface{})
	template := spec["template"].(map[string]interface{})
	podSpec := template["spec"].(map[string]interface{})
	tolerations, ok := podSpec["tolerations"].([]interface{})
	if !ok || len(tolerations) == 0 {
		t.Fatal("patched manifest should contain tolerations in pod spec")
	}

	// Check for control-plane toleration
	foundControlPlane := false
	foundMaster := false
	for _, t := range tolerations {
		tol, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		key, _ := tol["key"].(string)
		if key == "node-role.kubernetes.io/control-plane" {
			foundControlPlane = true
		}
		if key == "node-role.kubernetes.io/master" {
			foundMaster = true
		}
	}

	if !foundControlPlane {
		t.Error("patched manifest should contain toleration for node-role.kubernetes.io/control-plane")
	}
	if !foundMaster {
		t.Error("patched manifest should contain toleration for node-role.kubernetes.io/master")
	}
}

// TestKuredPatchManifest_MultiDocumentYAML verifies that multi-document YAML manifests
// are handled correctly (only DaemonSet resources are patched)
func TestKuredPatchManifest_MultiDocumentYAML(t *testing.T) {
	multiDoc := `apiVersion: v1
kind: ServiceAccount
metadata:
  name: kured
  namespace: kube-system
---
` + sampleKuredDaemonSet

	cfg := &config.Main{
		KubeconfigPath: "/tmp/kubeconfig",
		Addons: config.Addons{
			Kured: &config.Kured{
				Enabled:      true,
				ManifestURL:  "https://example.com/kured.yaml",
				KuredOptions: []string{"--period=30m"},
			},
		},
	}

	installer := NewKuredInstaller(cfg, nil)

	patched, err := installer.patchManifest(multiDoc)
	if err != nil {
		t.Fatalf("patchManifest returned error: %v", err)
	}

	// Both resources should be present
	if !strings.Contains(patched, "ServiceAccount") {
		t.Error("patched manifest should still contain the ServiceAccount resource")
	}
	if !strings.Contains(patched, "DaemonSet") {
		t.Error("patched manifest should still contain the DaemonSet resource")
	}

	// The kured option should appear in the patched manifest
	if !strings.Contains(patched, "--period=30m") {
		t.Error("patched manifest should contain the kured option --period=30m")
	}
}

// TestKuredDefaultConfig verifies that default values are set correctly
func TestKuredDefaultConfig(t *testing.T) {
	k := &config.Kured{}
	k.SetDefaults()

	if k.ManifestURL == "" {
		t.Error("default ManifestURL should not be empty")
	}
	if !strings.Contains(k.ManifestURL, "kured") {
		t.Errorf("default ManifestURL should reference kured, got: %s", k.ManifestURL)
	}
}
