// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNodePoolLabelSeparation(t *testing.T) {
	tests := []struct {
		name                    string
		yaml                    string
		expectedKubeLabels      int
		expectedKubeTaints      int
		expectedHetznerLabels   int
		expectKubeLabelKey      string
		expectKubeLabelValue    string
		expectHetznerLabelKey   string
		expectHetznerLabelValue string
	}{
		{
			name: "both kubernetes and hetzner labels",
			yaml: `
name: test-pool
instance_type: cpx22
kubernetes:
  labels:
    - key: workload
      value: database
    - key: env
      value: production
  taints:
    - key: dedicated
      value: database
      effect: NoSchedule
hetzner:
  labels:
    - key: backup
      value: "true"
    - key: isolated
      value: "true"
`,
			expectedKubeLabels:      2,
			expectedKubeTaints:      1,
			expectedHetznerLabels:   2,
			expectKubeLabelKey:      "workload",
			expectKubeLabelValue:    "database",
			expectHetznerLabelKey:   "backup",
			expectHetznerLabelValue: "true",
		},
		{
			name: "only kubernetes labels",
			yaml: `
name: test-pool
instance_type: cpx22
kubernetes:
  labels:
    - key: app
      value: web
`,
			expectedKubeLabels:    1,
			expectedKubeTaints:    0,
			expectedHetznerLabels: 0,
			expectKubeLabelKey:    "app",
			expectKubeLabelValue:  "web",
		},
		{
			name: "only hetzner labels",
			yaml: `
name: test-pool
instance_type: cpx22
hetzner:
  labels:
    - key: cost-center
      value: team-a
`,
			expectedKubeLabels:      0,
			expectedKubeTaints:      0,
			expectedHetznerLabels:   1,
			expectHetznerLabelKey:   "cost-center",
			expectHetznerLabelValue: "team-a",
		},
		{
			name: "no labels",
			yaml: `
name: test-pool
instance_type: cpx22
`,
			expectedKubeLabels:    0,
			expectedKubeTaints:    0,
			expectedHetznerLabels: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pool NodePool
			err := yaml.Unmarshal([]byte(tt.yaml), &pool)
			if err != nil {
				t.Fatalf("Failed to unmarshal YAML: %v", err)
			}

			// Test Kubernetes labels
			kubeLabels := pool.KubernetesLabels()
			if len(kubeLabels) != tt.expectedKubeLabels {
				t.Errorf("Expected %d Kubernetes labels, got %d", tt.expectedKubeLabels, len(kubeLabels))
			}

			if tt.expectedKubeLabels > 0 && tt.expectKubeLabelKey != "" {
				found := false
				for _, label := range kubeLabels {
					if label.Key == tt.expectKubeLabelKey && label.Value == tt.expectKubeLabelValue {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected Kubernetes label %s=%s not found", tt.expectKubeLabelKey, tt.expectKubeLabelValue)
				}
			}

			// Test Kubernetes taints
			kubeTaints := pool.KubernetesTaints()
			if len(kubeTaints) != tt.expectedKubeTaints {
				t.Errorf("Expected %d Kubernetes taints, got %d", tt.expectedKubeTaints, len(kubeTaints))
			}

			// Test Hetzner labels
			hetznerLabels := pool.HetznerLabels()
			if len(hetznerLabels) != tt.expectedHetznerLabels {
				t.Errorf("Expected %d Hetzner labels, got %d", tt.expectedHetznerLabels, len(hetznerLabels))
			}

			if tt.expectedHetznerLabels > 0 && tt.expectHetznerLabelKey != "" {
				found := false
				for _, label := range hetznerLabels {
					if label.Key == tt.expectHetznerLabelKey && label.Value == tt.expectHetznerLabelValue {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected Hetzner label %s=%s not found", tt.expectHetznerLabelKey, tt.expectHetznerLabelValue)
				}
			}
		})
	}
}

func TestWorkerNodePoolLabelHelpers(t *testing.T) {
	yamlConfig := `
name: database
instance_type: cpx22
kubernetes:
  labels:
    - key: workload
      value: database
  taints:
    - key: dedicated
      value: database
      effect: NoSchedule
hetzner:
  labels:
    - key: backup
      value: "true"
`

	var pool WorkerNodePool
	err := yaml.Unmarshal([]byte(yamlConfig), &pool)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	// Test KubernetesLabels helper
	kubeLabels := pool.KubernetesLabels()
	if len(kubeLabels) != 1 {
		t.Errorf("Expected 1 Kubernetes label, got %d", len(kubeLabels))
	}
	if kubeLabels[0].Key != "workload" || kubeLabels[0].Value != "database" {
		t.Errorf("Kubernetes label mismatch: got %s=%s", kubeLabels[0].Key, kubeLabels[0].Value)
	}

	// Test KubernetesTaints helper
	kubeTaints := pool.KubernetesTaints()
	if len(kubeTaints) != 1 {
		t.Errorf("Expected 1 Kubernetes taint, got %d", len(kubeTaints))
	}
	if kubeTaints[0].Key != "dedicated" || kubeTaints[0].Effect != "NoSchedule" {
		t.Errorf("Kubernetes taint mismatch: got %s:%s", kubeTaints[0].Key, kubeTaints[0].Effect)
	}

	// Test HetznerLabels helper
	hetznerLabels := pool.HetznerLabels()
	if len(hetznerLabels) != 1 {
		t.Errorf("Expected 1 Hetzner label, got %d", len(hetznerLabels))
	}
	if hetznerLabels[0].Key != "backup" || hetznerLabels[0].Value != "true" {
		t.Errorf("Hetzner label mismatch: got %s=%s", hetznerLabels[0].Key, hetznerLabels[0].Value)
	}
}
