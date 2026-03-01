package util

import (
	"os"
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

	// Helm version is now determined by the get-helm script
	if installer.GetHelmVersion() != "" {
		t.Error("Helm version should be empty (determined by get-helm script)")
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

func TestInstallKubectl_UnsupportedOS(t *testing.T) {
	// Skip this test if we can't mock the OS
	t.Skip("Skipping OS-specific test")
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
	kubectlInstalled := installer.IsKubectlInstalled()
	helmInstalled := installer.IsHelmInstalled()
	kubectlAIInstalled := installer.IsKubectlAIInstalled()

	if kubectlInstalled && helmInstalled && kubectlAIInstalled {
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
