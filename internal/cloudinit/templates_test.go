// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"strings"
	"testing"
)

func TestGenerateNATGatewayCloudInit(t *testing.T) {
	subnet := "10.0.0.0/16"
	cloudInit, err := GenerateNATGatewayCloudInit(subnet)
	if err != nil {
		t.Fatalf("Failed to generate NAT gateway cloud-init: %v", err)
	}

	// Verify the cloud-init contains the subnet
	if !strings.Contains(cloudInit, subnet) {
		t.Errorf("Expected cloud-init to contain subnet %s", subnet)
	}

	// Verify it contains IP forwarding configuration
	if !strings.Contains(cloudInit, "ip_forward") {
		t.Error("Expected cloud-init to contain ip_forward configuration")
	}

	// Verify it contains iptables MASQUERADE
	if !strings.Contains(cloudInit, "MASQUERADE") {
		t.Error("Expected cloud-init to contain MASQUERADE configuration")
	}
}

func TestGenerateK3sInstallMasterCommand(t *testing.T) {
	k3sVersion := "v1.28.5+k3s1"
	k3sToken := "test-token-123"
	baseArgs := "--cluster-init --tls-san=10.0.0.1"

	cmd, err := GenerateK3sInstallMasterCommand(k3sVersion, k3sToken, baseArgs)
	if err != nil {
		t.Fatalf("Failed to generate k3s install command: %v", err)
	}

	// Verify the command contains key elements
	if !strings.Contains(cmd, "curl -sfL https://get.k3s.io") {
		t.Error("Expected command to contain curl to get.k3s.io")
	}

	if !strings.Contains(cmd, k3sVersion) {
		t.Errorf("Expected command to contain k3s version %s", k3sVersion)
	}

	if !strings.Contains(cmd, k3sToken) {
		t.Errorf("Expected command to contain k3s token %s", k3sToken)
	}

	if !strings.Contains(cmd, baseArgs) {
		t.Errorf("Expected command to contain base args %s", baseArgs)
	}

	if !strings.Contains(cmd, "server") {
		t.Error("Expected command to contain 'server' for master installation")
	}

	// Verify it works with cluster-init flag (for parallel installation)
	if !strings.Contains(cmd, "--cluster-init") {
		t.Error("Expected command to contain --cluster-init flag for parallel master installation")
	}
}

func TestGenerateK3sInstallWorkerCommand(t *testing.T) {
	k3sVersion := "v1.28.5+k3s1"
	k3sToken := "test-token-123"
	k3sURL := "https://10.0.0.1:6443"
	baseArgs := " --flannel-iface=enp7s0"

	cmd, err := GenerateK3sInstallWorkerCommand(k3sVersion, k3sToken, k3sURL, baseArgs)
	if err != nil {
		t.Fatalf("Failed to generate k3s install command: %v", err)
	}

	// Verify the command contains key elements
	if !strings.Contains(cmd, "curl -sfL https://get.k3s.io") {
		t.Error("Expected command to contain curl to get.k3s.io")
	}

	if !strings.Contains(cmd, k3sVersion) {
		t.Errorf("Expected command to contain k3s version %s", k3sVersion)
	}

	if !strings.Contains(cmd, k3sToken) {
		t.Errorf("Expected command to contain k3s token %s", k3sToken)
	}

	if !strings.Contains(cmd, k3sURL) {
		t.Errorf("Expected command to contain k3s URL %s", k3sURL)
	}

	if !strings.Contains(cmd, baseArgs) {
		t.Errorf("Expected command to contain base args %s", baseArgs)
	}

	if !strings.Contains(cmd, "agent") {
		t.Error("Expected command to contain 'agent' for worker installation")
	}
}

func TestGenerateInternetConnectivityTestCommand(t *testing.T) {
	cmd, err := GenerateInternetConnectivityTestCommand()
	if err != nil {
		t.Fatalf("Failed to generate connectivity test command: %v", err)
	}

	// Verify the command contains key elements
	if !strings.Contains(cmd, "curl") {
		t.Error("Expected command to contain curl")
	}

	if !strings.Contains(cmd, "https://get.k3s.io") {
		t.Error("Expected command to test connectivity to get.k3s.io")
	}

	if !strings.Contains(cmd, "connected") {
		t.Error("Expected command to echo 'connected' on success")
	}
}
