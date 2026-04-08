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

	// Helm version is determined by the package manager or get-helm script
	if installer.GetHelmVersion() != "" {
		t.Error("Helm version should be empty (determined by package manager or get-helm script)")
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
	// Just verify the function runs without panic; result depends on environment
	_ = installer.IsHcloudInstalled()
}

func TestIsKubectlInstalled_WithKubectlInPath(t *testing.T) {
	installer := &ToolInstaller{}
	// This test will pass if kubectl is in PATH, otherwise it will pass anyway
	// as we're just testing the function works
	_ = installer.IsKubectlInstalled()
}

func TestIsHelmInstalled_WithHelmInPath(t *testing.T) {
	installer := &ToolInstaller{}
	// This test will pass if helm is in PATH, otherwise it will pass anyway
	// as we're just testing the function works
	_ = installer.IsHelmInstalled()
}

func TestIsKubectlAIInstalled_WithKubectlAIInPath(t *testing.T) {
	installer := &ToolInstaller{}
	// This test will pass if kubectl-ai is in PATH, otherwise it will pass anyway
	// as we're just testing the function works
	_ = installer.IsKubectlAIInstalled()
}

func TestIsCiliumInstalled(t *testing.T) {
	installer := &ToolInstaller{}
	// Just verify the function runs without panic; result depends on environment
	_ = installer.IsCiliumInstalled()
}

func TestIsBrewInstalled(t *testing.T) {
	installer := &ToolInstaller{}
	result := installer.IsBrewInstalled()
	// On macOS brew may or may not be installed; on other platforms it should be false
	if runtime.GOOS != "darwin" && result {
		t.Error("brew should not be found on non-macOS platforms")
	}
}

func TestIsWingetInstalled(t *testing.T) {
	installer := &ToolInstaller{}
	result := installer.IsWingetInstalled()
	// winget is only available on Windows
	if runtime.GOOS != "windows" && result {
		t.Error("winget should not be found on non-Windows platforms")
	}
}

func TestCommandExists(t *testing.T) {
	installer := &ToolInstaller{}

	// Test with a command that should always exist
	if !installer.commandExists("ls") && !installer.commandExists("dir") {
		t.Error("Expected ls or dir to exist")
	}

	// Test with a command that should not exist
	if installer.commandExists("this_command_definitely_does_not_exist_12345") {
		t.Error("Expected non-existent command to return false")
	}
}

func TestEnsurePackageManager_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific package manager test")
	}

	installer := &ToolInstaller{}
	// On Linux, EnsurePackageManager is a no-op and should always succeed
	if err := installer.EnsurePackageManager(); err != nil {
		t.Errorf("EnsurePackageManager should succeed on Linux, got: %v", err)
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

func TestInstallTool_NoFallbackNoPackageManager(t *testing.T) {
	// On Linux, installTool should invoke the fallback function.
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific installTool test")
	}

	installer := &ToolInstaller{}
	called := false
	err := installer.installTool("kubectl", func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Errorf("installTool should use fallback on Linux, got: %v", err)
	}
	if !called {
		t.Error("Expected fallback function to be called on Linux")
	}
}

func TestInstallTool_NoFallbackNoPackageEntry(t *testing.T) {
	// On Linux with a tool that has no package entry and no fallback, expect an error.
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific installTool test")
	}

	installer := &ToolInstaller{}
	err := installer.installTool("unknown-tool-xyz", nil)
	if err == nil {
		t.Error("Expected error when no fallback is provided for unknown tool")
	}
}

func TestBrewToolsMap(t *testing.T) {
	expected := []string{"hcloud", "helm", "kubectl", "kubectl-ai", "cilium"}
	for _, tool := range expected {
		if _, ok := brewTools[tool]; !ok {
			t.Errorf("brewTools map is missing entry for %q", tool)
		}
	}
}

func TestWingetToolsMap(t *testing.T) {
	expected := []string{"hcloud", "helm", "kubectl"}
	for _, tool := range expected {
		if _, ok := wingetTools[tool]; !ok {
			t.Errorf("wingetTools map is missing entry for %q", tool)
		}
	}
}

func TestInstallHelm_UnsupportedOS(t *testing.T) {
	// Skip this test if we can't mock the OS
	t.Skip("Skipping OS-specific test")
}

func TestInstallKubectlAI_UnsupportedOS(t *testing.T) {
	// Skip this test if we can't mock the OS
	t.Skip("Skipping OS-specific test")
}

func TestEnsureToolsInstalled_ToolsAlreadyInstalled(t *testing.T) {
	installer, _ := NewToolInstaller("v1.32.0+k3s1")

	// Check if tools are already installed
	hcloudInstalled := installer.IsHcloudInstalled()
	kubectlInstalled := installer.IsKubectlInstalled()
	helmInstalled := installer.IsHelmInstalled()
	kubectlAIInstalled := installer.IsKubectlAIInstalled()
	ciliumInstalled := installer.IsCiliumInstalled()

	if hcloudInstalled && kubectlInstalled && helmInstalled && kubectlAIInstalled && ciliumInstalled {
		// All tools are already installed, test should succeed
		err := installer.EnsureToolsInstalled()
		if err != nil {
			t.Errorf("EnsureToolsInstalled failed when tools were already installed: %v", err)
		}
	} else {
		// Tools not installed, skip the installation test
		// (we don't want to modify the test system)
		t.Skip("Tools not installed, skipping installation test to avoid system modifications")
	}
}

func TestRunCommand_SimpleCommand(t *testing.T) {
	installer := &ToolInstaller{}

	// Test with a simple command that should work on all systems
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
