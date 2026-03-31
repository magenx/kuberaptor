// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Main represents the main configuration structure
type Main struct {
	HetznerToken                      string           `yaml:"hetzner_token"`
	ClusterName                       string           `yaml:"cluster_name"`
	KubeconfigPath                    string           `yaml:"kubeconfig_path"`
	K3sVersion                        string           `yaml:"k3s_version"`
	Domain                            string           `yaml:"domain,omitempty"`
	APIServerHostname                 string           `yaml:"api_server_hostname,omitempty"`
	ScheduleWorkloadsOnMasters        bool             `yaml:"schedule_workloads_on_masters,omitempty"`
	MastersPool                       MasterNodePool   `yaml:"masters_pool"`
	WorkerNodePools                   []WorkerNodePool `yaml:"worker_node_pools,omitempty"`
	AdditionalPreK3sCommands          []string         `yaml:"additional_pre_k3s_commands,omitempty"`
	AdditionalPostK3sCommands         []string         `yaml:"additional_post_k3s_commands,omitempty"`
	AdditionalPackages                []string         `yaml:"additional_packages,omitempty"`
	KubeAPIServerArgs                 []string         `yaml:"kube_api_server_args,omitempty"`
	KubeSchedulerArgs                 []string         `yaml:"kube_scheduler_args,omitempty"`
	KubeControllerManagerArgs         []string         `yaml:"kube_controller_manager_args,omitempty"`
	KubeCloudControllerManagerArgs    []string         `yaml:"kube_cloud_controller_manager_args,omitempty"`
	ClusterAutoscalerArgs             []string         `yaml:"cluster_autoscaler_args,omitempty"`
	KubeletArgs                       []string         `yaml:"kubelet_args,omitempty"`
	KubeProxyArgs                     []string         `yaml:"kube_proxy_args,omitempty"`
	Image                             string           `yaml:"image,omitempty"`
	AutoscalingImage                  string           `yaml:"autoscaling_image,omitempty"`
	SnapshotOS                        string           `yaml:"snapshot_os,omitempty"`
	Networking                        Networking       `yaml:"networking,omitempty"`
	Datastore                         Datastore        `yaml:"datastore,omitempty"`
	Addons                            Addons           `yaml:"addons,omitempty"`
	LoadBalancer                      LoadBalancer     `yaml:"load_balancer,omitempty"`
	DNSZone                           DNSZone          `yaml:"dns_zone,omitempty"`
	SSLCertificate                    SSLCertificate   `yaml:"ssl_certificate,omitempty"`
	IncludeInstanceTypeInInstanceName bool             `yaml:"include_instance_type_in_instance_name,omitempty"`
	ProtectAgainstDeletion            bool             `yaml:"protect_against_deletion,omitempty"`
	APILoadBalancer                   APILoadBalancer  `yaml:"api_load_balancer,omitempty"`
	K3sUpgradeConcurrency             int64            `yaml:"k3s_upgrade_concurrency,omitempty"`
	GrowRootPartitionAutomatically    bool             `yaml:"grow_root_partition_automatically,omitempty"`
}

// SetDefaults sets default values for the configuration
func (c *Main) SetDefaults() {
	if c.Image == "" {
		c.Image = "ubuntu-24.04"
	}
	if c.SnapshotOS == "" {
		c.SnapshotOS = "default"
	}
	if c.K3sUpgradeConcurrency == 0 {
		c.K3sUpgradeConcurrency = 1
	}
	if c.HetznerToken == "" {
		c.HetznerToken = os.Getenv("HCLOUD_TOKEN")
	}
	// Set default to true for protect against deletion
	if !c.ProtectAgainstDeletion {
		c.ProtectAgainstDeletion = true
	}
	if !c.GrowRootPartitionAutomatically {
		c.GrowRootPartitionAutomatically = true
	}

	c.Networking.SetDefaults()
	c.Datastore.SetDefaults()
	c.Addons.SetDefaults()
	c.LoadBalancer.SetDefaults()
	c.DNSZone.SetDefaults()
	c.SSLCertificate.SetDefaults()
	c.APILoadBalancer.SetDefaults()

	// Set defaults for master and worker node pools
	c.MastersPool.SetDefaults()
	for i := range c.WorkerNodePools {
		c.WorkerNodePools[i].SetDefaults()
	}
}

// AllKubeletArgs returns all kubelet args including defaults
func (c *Main) AllKubeletArgs() []string {
	defaults := []string{"cloud-provider=external", "resolv-conf=/etc/k8s-resolv.conf"}
	return append(defaults, c.KubeletArgs...)
}

// ExpandPath expands a path with ~ to absolute path
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Expand ~ to home directory
	if path[:1] == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[1:])
	}

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return absPath, nil
}
