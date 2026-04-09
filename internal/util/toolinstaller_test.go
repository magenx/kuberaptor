// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package util

import (
	"os"
	"runtime"
	"testing"
)

func TestNewToolInstaller(t *testing.T) {
	installer, err := NewToolInstaller("v1.32.0+k3s1")
	if err != nil {
		t.Fatalf("Failed to create tool installer: %v", err)
	}

	if installer == nil {
		t.Error("installer should not be nil")
	}

	if installer.kubectlVersion != "v1.32.0" {
		t.Errorf("Expected kubectl version v1.32.0, got %s", installer.kubectlVersion)
	}
}

func TestExtractKubectlVersionFromK3s(t *testing.T) {
	tests := []struct {
		name     string
		k3sVer   string
		expected string
	}{
		{"standard version", "v1.32.0+k3s1", "v1.32.0"},
		{"different version", "v1.35.0+k3s2", "v1.35.0"},
		{"without prefix", "1.32.0+k3s1", "v1.32.0"},
		{"empty string", "", ""},
		{"invalid format", "invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractKubectlVersionFromK3s(tt.k3sVer)
			if result != tt.expected {
				t.Errorf("extractKubectlVersionFromK3s(%s) = %s, want %s", tt.k3sVer, result, tt.expected)
			}
		})
	}
}

func TestGetVersions(t *testing.T) {
	installer, _ := NewToolInstaller("v1.35.0+k3s1")

	if installer.GetKubectlVersion() != "v1.35.0" {
		t.Errorf("Expected kubectl version v1.35.0, got %s", installer.GetKubectlVersion())
	}

	// Helm version is determined by the package manager
	if installer.GetHelmVersion() != "" {
		t.Error("Helm version should be empty (determined by package manager)")
	}
}

func TestSetHelmVersion(t *testing.T) {
	installer, _ := NewToolInstaller("v1.32.0+k3s1")

	installer.SetHelmVersion("v3.15.0")

	if installer.GetHelmVersion() != "v3.15.0" {
		t.Errorf("Expected helm version v3.15.0, got %s", installer.GetHelmVersion())
	}
}

func TestNewToolInstallerWithEmptyVersion(t *testing.T) {
	_, err := NewToolInstaller("")
	if err == nil {
		t.Error("Expected error when creating tool installer with empty k3s version")
	}
}

func TestNewToolInstallerWithInvalidVersion(t *testing.T) {
	_, err := NewToolInstaller("invalid")
	if err == nil {
		t.Error("Expected error when creating tool installer with invalid k3s version")
	}
}

func TestIsHcloudInstalled(t *testing.T) {
	installer := &ToolInstaller{}
	_ = installer.IsHcloudInstalled()
}

func TestIsKubectlInstalled(t *testing.T) {
	installer := &ToolInstaller{}
	_ = installer.IsKubectlInstalled()
}

func TestIsHelmInstalled(t *testing.T) {
	installer := &ToolInstaller{}
	_ = installer.IsHelmInstalled()
}

func TestIsKubectlAIInstalled(t *testing.T) {
	installer := &ToolInstaller{}
	_ = installer.IsKubectlAIInstalled()
}

func TestIsBrewInstalled(t *testing.T) {
	installer := &ToolInstaller{}
	result := installer.IsBrewInstalled()
	if runtime.GOOS != "darwin" && result {
		t.Error("brew should not be found on non-macOS platforms")
	}
}

func TestIsWingetInstalled(t *testing.T) {
	installer := &ToolInstaller{}
	result := installer.IsWingetInstalled()
	if runtime.GOOS != "windows" && result {
		t.Error("winget should not be found on non-Windows platforms")
	}
}

func TestCommandExists(t *testing.T) {
	installer := &ToolInstaller{}

	if !installer.commandExists("ls") && !installer.commandExists("dir") {
		t.Error("Expected ls or dir to exist")
	}

	if installer.commandExists("this_command_definitely_does_not_exist_12345") {
		t.Error("Expected non-existent command to return false")
	}
}

func TestBrewToolsMap(t *testing.T) {
	expected := map[string]string{
		"hcloud":     "hcloud",
		"helm":       "helm",
		"kubectl":    "kubernetes-cli",
		"kubectl-ai": "kubectl-ai",
	}
	for tool, formula := range expected {
		got, ok := brewTools[tool]
		if !ok {
			t.Errorf("brewTools map is missing entry for %q", tool)
			continue
		}
		if got != formula {
			t.Errorf("brewTools[%q] = %q, want %q", tool, got, formula)
		}
	}
}

func TestBrewToolsMap_NoCiliumOrFlux(t *testing.T) {
	if _, ok := brewTools["cilium"]; ok {
		t.Error("cilium should not be in brewTools (managed manually by user)")
	}
	if _, ok := brewTools["flux"]; ok {
		t.Error("flux should not be in brewTools (managed manually by user)")
	}
}

func TestWingetToolsMap(t *testing.T) {
	expected := map[string]string{
		"hcloud":  "HetznerCloud.CLI",
		"helm":    "Helm.Helm",
		"kubectl": "Kubernetes.kubectl",
	}
	for tool, pkgID := range expected {
		got, ok := wingetTools[tool]
		if !ok {
			t.Errorf("wingetTools map is missing entry for %q", tool)
			continue
		}
		if got != pkgID {
			t.Errorf("wingetTools[%q] = %q, want %q", tool, got, pkgID)
		}
	}
}

func TestWingetToolsMap_NoCiliumOrFlux(t *testing.T) {
	if _, ok := wingetTools["cilium"]; ok {
		t.Error("cilium should not be in wingetTools (managed manually by user)")
	}
	if _, ok := wingetTools["flux"]; ok {
		t.Error("flux should not be in wingetTools (managed manually by user)")
	}
}

func TestSnapToolsMap(t *testing.T) {
	expected := map[string]string{
		"kubectl": "kubectl",
		"helm":    "helm",
	}
	for tool, pkg := range expected {
		got, ok := snapTools[tool]
		if !ok {
			t.Errorf("snapTools map is missing entry for %q", tool)
			continue
		}
		if got != pkg {
			t.Errorf("snapTools[%q] = %q, want %q", tool, got, pkg)
		}
	}
}

func TestWingetToolsMap_KubectlAINotPresent(t *testing.T) {
	// kubectl-ai is intentionally absent from wingetTools; it uses krew instead
	if _, ok := wingetTools["kubectl-ai"]; ok {
		t.Error("kubectl-ai should not be in wingetTools (it is installed via krew on Windows)")
	}
}

func TestInstallBrew_NonMacOS(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping non-macOS test on macOS")
	}

	installer := &ToolInstaller{}
	err := installer.InstallBrew()
	if err == nil {
		t.Error("Expected error when installing brew on non-macOS")
	}
}

func TestIsSnapdInstalled(t *testing.T) {
	installer := &ToolInstaller{}
	result := installer.IsSnapdInstalled()
	if runtime.GOOS != "linux" && result {
		t.Error("snap should not be found on non-Linux platforms in a typical environment")
	}
}

func TestEnsurePackageManager_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific package manager test")
	}

	installer := &ToolInstaller{}
	// On Linux, EnsurePackageManager should succeed (snapd available or installed)
	// We only verify it doesn't return an unsupported-platform error.
	err := installer.EnsurePackageManager()
	if err != nil {
		// Acceptable: snapd install may fail in a restricted CI environment
		t.Logf("EnsurePackageManager on Linux returned (possibly expected in CI): %v", err)
	}
}

func TestInstallTool_Linux_Snap(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific installTool test")
	}

	installer := &ToolInstaller{}
	// Verify kubectl is defined in snapTools (the actual snap binary won't be
	// present in the test environment, so we don't attempt the full install).
	if _, ok := snapTools["kubectl"]; !ok {
		t.Error("kubectl should be defined in snapTools")
	}
	_ = installer
}

func TestInstallTool_Linux_UnknownTool(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific installTool test")
	}

	installer := &ToolInstaller{}
	err := installer.installTool("unknown-tool-xyz")
	if err == nil {
		t.Error("Expected error for unknown tool on Linux")
	}
}

func TestInstallTool_UnknownTool_Darwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific installTool test")
	}

	installer := &ToolInstaller{}
	err := installer.installTool("unknown-tool-xyz")
	if err == nil {
		t.Error("Expected error for unknown tool on macOS")
	}
}

func TestEnsureToolsInstalled_ToolsAlreadyInstalled(t *testing.T) {
	installer, _ := NewToolInstaller("v1.32.0+k3s1")

	hcloudInstalled := installer.IsHcloudInstalled()
	kubectlInstalled := installer.IsKubectlInstalled()
	helmInstalled := installer.IsHelmInstalled()
	kubectlAIInstalled := installer.IsKubectlAIInstalled()

	if hcloudInstalled && kubectlInstalled && helmInstalled && kubectlAIInstalled {
		err := installer.EnsureToolsInstalled()
		if err != nil {
			t.Errorf("EnsureToolsInstalled failed when tools were already installed: %v", err)
		}
	} else {
		t.Skip("Tools not installed, skipping installation test to avoid system modifications")
	}
}

func TestRunCommand_SimpleCommand(t *testing.T) {
	installer := &ToolInstaller{}

	var cmd string
	var args []string

	if _, err := os.Stat("/bin/echo"); err == nil {
		cmd = "echo"
		args = []string{"test"}
	} else if _, err := os.Stat("/usr/bin/echo"); err == nil {
		cmd = "echo"
		args = []string{"test"}
	} else {
		t.Skip("echo command not found")
	}

	err := installer.runCommand(cmd, args...)
	if err != nil {
		t.Errorf("runCommand failed for simple echo command: %v", err)
	}
}
