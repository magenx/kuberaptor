// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package util

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// ToolInstaller handles detection and installation of required tools
type ToolInstaller struct {
	kubectlVersion string
	helmVersion    string
	ciliumVersion  string
}

// NewToolInstaller creates a new tool installer with versions
func NewToolInstaller(k3sVersion string) (*ToolInstaller, error) {
	kubectlVersion := extractKubectlVersionFromK3s(k3sVersion)
	if kubectlVersion == "" {
		return nil, fmt.Errorf("unable to extract kubectl version from k3s version: %s", k3sVersion)
	}

	return &ToolInstaller{
		kubectlVersion: kubectlVersion,
		helmVersion:    "", // Helm version determined by get-helm script
		ciliumVersion:  "", // Cilium CLI version determined by stable.txt
	}, nil
}

// extractKubectlVersionFromK3s extracts the Kubernetes version from k3s version
// Example: v1.35.0+k3s1 -> v1.35.0
func extractKubectlVersionFromK3s(k3sVersion string) string {
	if k3sVersion == "" {
		return ""
	}

	// Match pattern vX.Y.Z+k3sN and extract vX.Y.Z
	re := regexp.MustCompile(`^(v?\d+\.\d+\.\d+)`)
	matches := re.FindStringSubmatch(k3sVersion)

	if len(matches) > 1 {
		version := matches[1]
		// Ensure it has the 'v' prefix
		if !strings.HasPrefix(version, "v") {
			version = "v" + version
		}
		return version
	}

	return ""
}

// SetHelmVersion allows setting a custom helm version
func (t *ToolInstaller) SetHelmVersion(version string) {
	t.helmVersion = version
}

// SetCiliumVersion allows setting a custom cilium CLI version
func (t *ToolInstaller) SetCiliumVersion(version string) {
	t.ciliumVersion = version
}

// GetKubectlVersion returns the kubectl version that will be installed
func (t *ToolInstaller) GetKubectlVersion() string {
	return t.kubectlVersion
}

// GetHelmVersion returns the helm version that will be installed
func (t *ToolInstaller) GetHelmVersion() string {
	return t.helmVersion
}

// GetCiliumVersion returns the cilium CLI version that will be installed
func (t *ToolInstaller) GetCiliumVersion() string {
	return t.ciliumVersion
}

// IsKubectlInstalled checks if kubectl is available
func (t *ToolInstaller) IsKubectlInstalled() bool {
	_, err := exec.LookPath("kubectl")
	return err == nil
}

// IsHelmInstalled checks if helm is available
func (t *ToolInstaller) IsHelmInstalled() bool {
	_, err := exec.LookPath("helm")
	return err == nil
}

// IsKubectlAIInstalled checks if kubectl-ai is available
func (t *ToolInstaller) IsKubectlAIInstalled() bool {
	_, err := exec.LookPath("kubectl-ai")
	return err == nil
}

// IsCiliumInstalled checks if cilium CLI is available
func (t *ToolInstaller) IsCiliumInstalled() bool {
	_, err := exec.LookPath("cilium")
	return err == nil
}

// InstallKubectl installs kubectl globally using direct download
func (t *ToolInstaller) InstallKubectl() error {
	fmt.Printf("Installing kubectl %s globally...\n", t.kubectlVersion)

	osName := runtime.GOOS
	if osName != "linux" && osName != "darwin" {
		return fmt.Errorf("unsupported operating system: %s", osName)
	}

	// Validate architecture (supported: amd64, arm64)
	arch := runtime.GOARCH
	if arch != "amd64" && arch != "arm64" {
		return fmt.Errorf("unsupported architecture: %s", arch)
	}

	fmt.Printf("Downloading kubectl %s for %s/%s...\n", t.kubectlVersion, osName, arch)

	// Download kubectl binary
	kubectlURL := fmt.Sprintf("https://dl.k8s.io/release/%s/bin/%s/%s/kubectl", t.kubectlVersion, osName, arch)
	if err := t.runCommand("curl", "-fsSLO", kubectlURL); err != nil {
		return fmt.Errorf("failed to download kubectl: %w", err)
	}

	// Download checksum
	checksumURL := fmt.Sprintf("https://dl.k8s.io/release/%s/bin/%s/%s/kubectl.sha256", t.kubectlVersion, osName, arch)
	if err := t.runCommand("curl", "-fsSLO", checksumURL); err != nil {
		return fmt.Errorf("failed to download kubectl checksum: %w", err)
	}

	// Verify checksum (command differs between Linux and macOS)
	fmt.Println("Verifying kubectl checksum")
	var checksumCmd string
	if osName == "linux" {
		checksumCmd = "echo \"$(cat kubectl.sha256)  kubectl\" | sha256sum --check"
	} else if osName == "darwin" {
		checksumCmd = "echo \"$(cat kubectl.sha256)  kubectl\" | shasum -a 256 --check"
	} else {
		// Should never reach here due to OS validation above
		return fmt.Errorf("unsupported operating system: %s", osName)
	}

	cmd := exec.Command("bash", "-c", checksumCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up downloaded files
		os.Remove("kubectl")
		os.Remove("kubectl.sha256")
		return fmt.Errorf("kubectl checksum verification failed: %w\nOutput: %s", err, string(output))
	}

	// Make executable and move to /usr/local/bin
	if err := os.Chmod("kubectl", 0755); err != nil {
		os.Remove("kubectl")
		os.Remove("kubectl.sha256")
		return fmt.Errorf("failed to make kubectl executable: %w", err)
	}

	if err := t.runCommand("sudo", "mv", "kubectl", "/usr/local/bin/kubectl"); err != nil {
		os.Remove("kubectl")
		os.Remove("kubectl.sha256")
		return fmt.Errorf("failed to move kubectl to /usr/local/bin: %w", err)
	}

	// Clean up checksum file
	os.Remove("kubectl.sha256")

	fmt.Println("✓ kubectl installed successfully to /usr/local/bin/kubectl")
	return nil
}

// installFromScript is a generic helper to install tools from shell scripts
// It handles download, execution, and cleanup of installation scripts
func (t *ToolInstaller) installFromScript(toolName, scriptURL, scriptFile, executor string) error {
	osName := runtime.GOOS
	if osName != "linux" && osName != "darwin" {
		return fmt.Errorf("unsupported operating system: %s", osName)
	}

	fmt.Printf("Downloading %s installation script\n", toolName)

	// Download the installation script
	if err := t.runCommand("curl", "-fsSL", "-o", scriptFile, scriptURL); err != nil {
		return fmt.Errorf("failed to download %s install script: %w", toolName, err)
	}

	// Make the script executable
	if err := os.Chmod(scriptFile, 0700); err != nil {
		os.Remove(scriptFile)
		return fmt.Errorf("failed to make %s script executable: %w", toolName, err)
	}

	// Execute the script
	fmt.Printf("Running %s installation script\n", toolName)
	var cmd *exec.Cmd
	if executor == "" || executor == "direct" {
		cmd = exec.Command("./" + scriptFile)
	} else {
		cmd = exec.Command(executor, scriptFile)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Remove(scriptFile)
		return fmt.Errorf("failed to run %s install script: %w", toolName, err)
	}

	// Clean up the script
	os.Remove(scriptFile)

	fmt.Printf("✓ %s installed successfully\n", toolName)
	return nil
}

// InstallHelm installs helm globally using the official get-helm-4 script
// The script automatically detects OS and architecture
func (t *ToolInstaller) InstallHelm() error {
	fmt.Println("Installing helm globally")
	return t.installFromScript(
		"helm",
		"https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-4",
		"get_helm.sh",
		"direct",
	)
}

// InstallKubectlAI installs kubectl-ai globally using the official installation script
// The script automatically detects OS and architecture
func (t *ToolInstaller) InstallKubectlAI() error {
	fmt.Println("Installing kubectl-ai globally")
	return t.installFromScript(
		"kubectl-ai",
		"https://raw.githubusercontent.com/GoogleCloudPlatform/kubectl-ai/main/install.sh",
		"install_kubectl_ai.sh",
		"bash",
	)
}

// InstallCilium installs cilium CLI globally using the official installation method
// The installation automatically detects OS and architecture
func (t *ToolInstaller) InstallCilium() error {
	fmt.Println("Installing cilium CLI globally")

	osName := runtime.GOOS
	if osName != "linux" && osName != "darwin" {
		return fmt.Errorf("unsupported operating system: %s", osName)
	}

	// Validate architecture (supported: amd64, arm64)
	arch := runtime.GOARCH
	if arch != "amd64" && arch != "arm64" {
		return fmt.Errorf("unsupported architecture: %s", arch)
	}

	fmt.Println("Determining latest cilium CLI version")

	// Get the stable version from cilium-cli repository
	var ciliumVersion string
	if t.ciliumVersion != "" {
		ciliumVersion = t.ciliumVersion
	} else {
		// Fetch stable version from GitHub
		versionURL := "https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt"
		cmd := exec.Command("curl", "-fsS", versionURL)
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to fetch cilium CLI version: %w", err)
		}
		ciliumVersion = strings.TrimSpace(string(output))
	}

	fmt.Printf("Downloading cilium CLI %s for %s/%s...\n", ciliumVersion, osName, arch)

	// Build download URL
	tarballName := fmt.Sprintf("cilium-%s-%s.tar.gz", osName, arch)
	checksumName := fmt.Sprintf("%s.sha256sum", tarballName)
	baseURL := fmt.Sprintf("https://github.com/cilium/cilium-cli/releases/download/%s", ciliumVersion)

	// Download tarball
	if err := t.runCommand("curl", "-fsLO",
		fmt.Sprintf("%s/%s", baseURL, tarballName),
		fmt.Sprintf("%s/%s", baseURL, checksumName)); err != nil {
		return fmt.Errorf("failed to download cilium CLI: %w", err)
	}

	// Verify checksum
	fmt.Println("Verifying cilium CLI checksum")
	cmd := exec.Command("sha256sum", "--check", checksumName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up downloaded files
		os.Remove(tarballName)
		os.Remove(checksumName)
		return fmt.Errorf("cilium CLI checksum verification failed: %w\nOutput: %s", err, string(output))
	}

	// Extract and install
	if err := t.runCommand("sudo", "tar", "xzf", tarballName, "-C", "/usr/local/bin"); err != nil {
		os.Remove(tarballName)
		os.Remove(checksumName)
		return fmt.Errorf("failed to extract cilium CLI: %w", err)
	}

	// Clean up downloaded files
	os.Remove(tarballName)
	os.Remove(checksumName)

	fmt.Println("✓ cilium CLI installed successfully to /usr/local/bin/cilium")
	return nil
}

// EnsureToolsInstalled checks and installs kubectl, helm, kubectl-ai, and cilium CLI if needed
// Tools are installed in the following order:
// 1. kubectl - Kubernetes command-line tool
// 2. helm - Kubernetes package manager
// 3. kubectl-ai - AI-powered kubectl assistant
// 4. cilium - Cilium CNI CLI tool
func (t *ToolInstaller) EnsureToolsInstalled() error {
	var errors []string

	if !t.IsKubectlInstalled() {
		if err := t.InstallKubectl(); err != nil {
			errors = append(errors, fmt.Sprintf("kubectl: %v", err))
		}
	} else {
		fmt.Println("✓ kubectl is already installed")
	}

	if !t.IsHelmInstalled() {
		if err := t.InstallHelm(); err != nil {
			errors = append(errors, fmt.Sprintf("helm: %v", err))
		}
	} else {
		fmt.Println("✓ helm is already installed")
	}

	if !t.IsKubectlAIInstalled() {
		if err := t.InstallKubectlAI(); err != nil {
			errors = append(errors, fmt.Sprintf("kubectl-ai: %v", err))
		}
	} else {
		fmt.Println("✓ kubectl-ai is already installed")
	}

	if !t.IsCiliumInstalled() {
		if err := t.InstallCilium(); err != nil {
			errors = append(errors, fmt.Sprintf("cilium: %v", err))
		}
	} else {
		fmt.Println("✓ cilium CLI is already installed")
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to install tools:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// commandExists checks if a command is available in PATH
func (t *ToolInstaller) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// runCommand executes a command and shows output
func (t *ToolInstaller) runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Print output if there's any
	if stdout.Len() > 0 {
		fmt.Print(stdout.String())
	}
	if stderr.Len() > 0 {
		fmt.Fprint(os.Stderr, stderr.String())
	}

	return err
}
