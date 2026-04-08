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

// brewTools maps logical tool names to their Homebrew formula names.
var brewTools = map[string]string{
	"hcloud":     "hcloud",
	"helm":       "helm",
	"kubectl":    "kubectl",
	"kubectl-ai": "kubectl-ai",
	"cilium":     "cilium-cli",
}

// wingetTools maps logical tool names to their winget package IDs.
// Tools absent from this map fall back to script-based installation.
var wingetTools = map[string]string{
	"hcloud":  "Hetzner.hcloud",
	"helm":    "Helm.Helm",
	"kubectl": "Kubernetes.kubectl",
}

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
		helmVersion:    "", // Helm version determined by package manager or get-helm script
		ciliumVersion:  "", // Cilium CLI version determined by package manager or stable.txt
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

// IsHcloudInstalled checks if the hcloud CLI is available
func (t *ToolInstaller) IsHcloudInstalled() bool {
	_, err := exec.LookPath("hcloud")
	return err == nil
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

// IsBrewInstalled checks if Homebrew is available (macOS)
func (t *ToolInstaller) IsBrewInstalled() bool {
	_, err := exec.LookPath("brew")
	return err == nil
}

// IsWingetInstalled checks if winget is available (Windows)
func (t *ToolInstaller) IsWingetInstalled() bool {
	_, err := exec.LookPath("winget")
	return err == nil
}

// InstallBrew installs Homebrew on macOS using the official installation script.
// The script is fetched from the official Homebrew GitHub repository.
func (t *ToolInstaller) InstallBrew() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("Homebrew installation is only supported on macOS")
	}

	fmt.Println("Installing Homebrew package manager...")
	return t.installFromScript(
		"Homebrew",
		"https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh",
		"homebrew_install.sh",
		"bash",
	)
}

// EnsurePackageManager detects the current OS and ensures the appropriate
// package manager (brew on macOS, winget on Windows) is available.
// On Linux, no package manager setup is performed.
// Returns an error only if the package manager cannot be installed.
func (t *ToolInstaller) EnsurePackageManager() error {
	switch runtime.GOOS {
	case "darwin":
		if !t.IsBrewInstalled() {
			fmt.Println("Homebrew not found. Installing Homebrew...")
			if err := t.InstallBrew(); err != nil {
				return fmt.Errorf("failed to install Homebrew: %w", err)
			}
			fmt.Println("✓ Homebrew installed successfully")
		} else {
			fmt.Println("✓ Homebrew is already installed")
		}
	case "windows":
		if !t.IsWingetInstalled() {
			return fmt.Errorf("winget is not available on this system. " +
				"Please install Windows Package Manager (winget) from " +
				"https://aka.ms/getwinget and re-run this command")
		}
		fmt.Println("✓ winget is already installed")
	default:
		// Linux and other platforms: no package manager setup required
	}
	return nil
}

// installWithBrew installs a package using Homebrew.
func (t *ToolInstaller) installWithBrew(packageName string) error {
	fmt.Printf("Installing %s via Homebrew...\n", packageName)
	if err := t.runCommand("brew", "install", packageName); err != nil {
		return fmt.Errorf("brew install %s failed: %w", packageName, err)
	}
	fmt.Printf("✓ %s installed successfully via Homebrew\n", packageName)
	return nil
}

// installWithWinget installs a package using winget.
func (t *ToolInstaller) installWithWinget(packageID string) error {
	fmt.Printf("Installing %s via winget...\n", packageID)
	err := t.runCommand("winget", "install",
		"--id", packageID,
		"--silent",
		"--accept-package-agreements",
		"--accept-source-agreements",
	)
	if err != nil {
		return fmt.Errorf("winget install %s failed: %w", packageID, err)
	}
	fmt.Printf("✓ %s installed successfully via winget\n", packageID)
	return nil
}

// installTool installs a named tool using the appropriate package manager for
// the current OS, falling back to the provided fallbackFn on Linux or when the
// tool has no package-manager entry.
func (t *ToolInstaller) installTool(toolName string, fallbackFn func() error) error {
	switch runtime.GOOS {
	case "darwin":
		if formula, ok := brewTools[toolName]; ok {
			return t.installWithBrew(formula)
		}
		// No brew formula defined — fall through to fallback
	case "windows":
		if pkgID, ok := wingetTools[toolName]; ok {
			return t.installWithWinget(pkgID)
		}
		// No winget entry defined — fall through to fallback
	}

	// Linux, or tools not covered by a package manager entry
	if fallbackFn != nil {
		return fallbackFn()
	}
	return fmt.Errorf("no installation method available for %s on %s", toolName, runtime.GOOS)
}

// InstallHcloud installs the hcloud CLI.
// On macOS it uses Homebrew; on Windows it uses winget; on Linux it falls back
// to the official install script.
func (t *ToolInstaller) InstallHcloud() error {
	fmt.Println("Installing hcloud CLI...")
	return t.installTool("hcloud", func() error {
		return t.installFromScript(
			"hcloud",
			"https://raw.githubusercontent.com/hetznercloud/cli/main/install.sh",
			"install_hcloud.sh",
			"bash",
		)
	})
}

// InstallKubectl installs kubectl.
// On macOS it uses Homebrew; on Windows it uses winget; on Linux it falls back
// to the direct binary download from dl.k8s.io.
func (t *ToolInstaller) InstallKubectl() error {
	fmt.Printf("Installing kubectl %s...\n", t.kubectlVersion)
	return t.installTool("kubectl", func() error {
		return t.installKubectlLinux()
	})
}

// installKubectlLinux performs a direct binary download of kubectl for Linux.
func (t *ToolInstaller) installKubectlLinux() error {
	osName := runtime.GOOS
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
		os.Remove("kubectl") // best-effort cleanup; ignore removal errors
		return fmt.Errorf("failed to download kubectl checksum: %w", err)
	}

	// Verify checksum
	fmt.Println("Verifying kubectl checksum")
	cmd := exec.Command("bash", "-c", "echo \"$(cat kubectl.sha256)  kubectl\" | sha256sum --check")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// best-effort cleanup; ignore removal errors
		os.Remove("kubectl")
		os.Remove("kubectl.sha256")
		return fmt.Errorf("kubectl checksum verification failed: %w\nOutput: %s", err, string(output))
	}

	// Make executable and move to /usr/local/bin
	if err := os.Chmod("kubectl", 0755); err != nil {
		// best-effort cleanup; ignore removal errors
		os.Remove("kubectl")
		os.Remove("kubectl.sha256")
		return fmt.Errorf("failed to make kubectl executable: %w", err)
	}

	if err := t.runCommand("sudo", "mv", "kubectl", "/usr/local/bin/kubectl"); err != nil {
		// best-effort cleanup; ignore removal errors
		os.Remove("kubectl")
		os.Remove("kubectl.sha256")
		return fmt.Errorf("failed to move kubectl to /usr/local/bin: %w", err)
	}

	os.Remove("kubectl.sha256") // best-effort cleanup; ignore removal errors
	fmt.Println("✓ kubectl installed successfully to /usr/local/bin/kubectl")
	return nil
}

// installFromScript is a generic helper to install tools from shell scripts.
// It handles download, execution, and cleanup of installation scripts.
func (t *ToolInstaller) installFromScript(toolName, scriptURL, scriptFile, executor string) error {
	osName := runtime.GOOS
	if osName == "windows" {
		return fmt.Errorf("script-based installation is not supported on Windows for %s; please install it manually", toolName)
	}

	fmt.Printf("Downloading %s installation script\n", toolName)

	// Download the installation script
	if err := t.runCommand("curl", "-fsSL", "-o", scriptFile, scriptURL); err != nil {
		return fmt.Errorf("failed to download %s install script: %w", toolName, err)
	}

	// Make the script executable
	if err := os.Chmod(scriptFile, 0700); err != nil {
		os.Remove(scriptFile) // best-effort cleanup; ignore removal errors
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
		os.Remove(scriptFile) // best-effort cleanup; ignore removal errors
		return fmt.Errorf("failed to run %s install script: %w", toolName, err)
	}

	// Clean up the script
	os.Remove(scriptFile) // best-effort cleanup; ignore removal errors

	fmt.Printf("✓ %s installed successfully\n", toolName)
	return nil
}

// InstallHelm installs Helm.
// On macOS it uses Homebrew; on Windows it uses winget; on Linux it falls back
// to the official get-helm-4 script.
func (t *ToolInstaller) InstallHelm() error {
	fmt.Println("Installing helm...")
	return t.installTool("helm", func() error {
		return t.installFromScript(
			"helm",
			"https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-4",
			"get_helm.sh",
			"direct",
		)
	})
}

// InstallKubectlAI installs kubectl-ai.
// On macOS it uses Homebrew; on other platforms it falls back to the official
// installation script (winget does not currently carry kubectl-ai).
func (t *ToolInstaller) InstallKubectlAI() error {
	fmt.Println("Installing kubectl-ai...")
	return t.installTool("kubectl-ai", func() error {
		return t.installFromScript(
			"kubectl-ai",
			"https://raw.githubusercontent.com/GoogleCloudPlatform/kubectl-ai/main/install.sh",
			"install_kubectl_ai.sh",
			"bash",
		)
	})
}

// InstallCilium installs the Cilium CLI.
// On macOS it uses Homebrew; on other platforms it falls back to the official
// tarball download from GitHub (winget does not currently carry cilium-cli).
func (t *ToolInstaller) InstallCilium() error {
	fmt.Println("Installing cilium CLI...")
	return t.installTool("cilium", func() error {
		return t.installCiliumLinux()
	})
}

// installCiliumLinux performs a tarball download of the Cilium CLI for Linux.
func (t *ToolInstaller) installCiliumLinux() error {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	if arch != "amd64" && arch != "arm64" {
		return fmt.Errorf("unsupported architecture: %s", arch)
	}

	fmt.Println("Determining latest cilium CLI version")

	var ciliumVersion string
	if t.ciliumVersion != "" {
		ciliumVersion = t.ciliumVersion
	} else {
		versionURL := "https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt"
		cmd := exec.Command("curl", "-fsS", versionURL)
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to fetch cilium CLI version: %w", err)
		}
		ciliumVersion = strings.TrimSpace(string(output))
	}

	fmt.Printf("Downloading cilium CLI %s for %s/%s...\n", ciliumVersion, osName, arch)

	tarballName := fmt.Sprintf("cilium-%s-%s.tar.gz", osName, arch)
	checksumName := fmt.Sprintf("%s.sha256sum", tarballName)
	baseURL := fmt.Sprintf("https://github.com/cilium/cilium-cli/releases/download/%s", ciliumVersion)

	if err := t.runCommand("curl", "-fsSL", "--remote-name-all",
		fmt.Sprintf("%s/%s", baseURL, tarballName),
		fmt.Sprintf("%s/%s", baseURL, checksumName)); err != nil {
		return fmt.Errorf("failed to download cilium CLI: %w", err)
	}

	fmt.Println("Verifying cilium CLI checksum")
	cmd := exec.Command("sha256sum", "--check", checksumName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// best-effort cleanup; ignore removal errors
		os.Remove(tarballName)
		os.Remove(checksumName)
		return fmt.Errorf("cilium CLI checksum verification failed: %w\nOutput: %s", err, string(output))
	}

	if err := t.runCommand("sudo", "tar", "xzf", tarballName, "-C", "/usr/local/bin"); err != nil {
		// best-effort cleanup; ignore removal errors
		os.Remove(tarballName)
		os.Remove(checksumName)
		return fmt.Errorf("failed to extract cilium CLI: %w", err)
	}

	// best-effort cleanup; ignore removal errors
	os.Remove(tarballName)
	os.Remove(checksumName)
	fmt.Println("✓ cilium CLI installed successfully to /usr/local/bin/cilium")
	return nil
}

// EnsureToolsInstalled ensures the package manager is available and then checks
// and installs hcloud CLI, kubectl, helm, kubectl-ai, and cilium CLI if needed.
// Tools are installed in the following order:
// 1. Package manager (brew on macOS, winget on Windows)
// 2. hcloud  - Hetzner Cloud CLI
// 3. kubectl - Kubernetes command-line tool
// 4. helm    - Kubernetes package manager
// 5. kubectl-ai - AI-powered kubectl assistant
// 6. cilium  - Cilium CNI CLI tool
func (t *ToolInstaller) EnsureToolsInstalled() error {
	// Ensure the appropriate package manager is available first
	if err := t.EnsurePackageManager(); err != nil {
		return fmt.Errorf("package manager setup failed: %w", err)
	}

	var errors []string

	if !t.IsHcloudInstalled() {
		if err := t.InstallHcloud(); err != nil {
			errors = append(errors, fmt.Sprintf("hcloud: %v", err))
		}
	} else {
		fmt.Println("✓ hcloud CLI is already installed")
	}

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
