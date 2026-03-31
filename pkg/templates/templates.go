// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package templates

import (
	"bytes"
	"fmt"
	"text/template"
)

// CloudInitData holds data for cloud-init template
type CloudInitData struct {
	GrowPartStr           string
	GrowRootDisabledFile  string
	Eth1Str               string
	FirewallFiles         string
	SSHFiles              string
	InitFiles             string
	PackagesStr           string
	PostCreateCommandsStr string
}

// MasterInstallData holds data for master install script
type MasterInstallData struct {
	PrivateNetworkEnabled         string
	PrivateNetworkSubnet          string
	CNI                           string
	CNIMode                       string
	FlannelBackend                string
	EmbeddedRegistryMirrorEnabled string
	LocalPathStorageClassEnabled  string
	TraefikEnabled                string
	ServiceLBEnabled              string
	MetricsServerEnabled          string
	K3sVersion                    string
	K3sToken                      string
	ClusterCIDR                   string
	ServiceCIDR                   string
	ClusterDNS                    string
	APIServerHostname             string
	LoadBalancerIP                string
	ScheduleWorkloadsOnMasters    string
	KubeAPIServerArgs             string
	KubeSchedulerArgs             string
	KubeControllerManagerArgs     string
	KubeletArgs                   string
	KubeProxyArgs                 string
	AdditionalPreK3sCommands      string
	AdditionalPostK3sCommands     string
}

// WorkerInstallData holds data for worker install script
type WorkerInstallData struct {
	PrivateNetworkEnabled     string
	PrivateNetworkSubnet      string
	CNI                       string
	CNIMode                   string
	FlannelBackend            string
	K3sVersion                string
	K3sToken                  string
	FirstMasterPrivateIP      string
	KubeletArgs               string
	KubeProxyArgs             string
	NodeLabels                string
	NodeTaints                string
	AdditionalPreK3sCommands  string
	AdditionalPostK3sCommands string
}

// RenderTemplate renders a template with the given data
func RenderTemplate(tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New("template").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
