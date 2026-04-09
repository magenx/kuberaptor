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

// brewTools maps logical tool names to their Homebrew formula names (macOS only).
var brewTools = map[string]string{
	"hcloud":     "hcloud",
	"helm":       "helm",
	"kubectl":    "kubernetes-cli",
	"kubectl-ai": "kubectl-ai",
}

// wingetTools maps logical tool names to their winget package IDs.
// kubectl-ai is handled separately via krew (see InstallKubectlAI).
var wingetTools = map[string]string{
	"hcloud":  "HetznerCloud.CLI",
	"helm":    "Helm.Helm",
	"kubectl": "Kubernetes.kubectl",
}

// snapTools maps logical tool names to their snap package names on Linux.
var snapTools = map[string]string{
	"kubectl": "kubectl",
	"helm":    "helm",
}

// ToolInstaller handles detection and installation of required tools
type ToolInstaller struct {
	kubectlVersion string
	helmVersion    string
}

// NewToolInstaller creates a new tool installer with versions
func NewToolInstaller(k3sVersion string) (*ToolInstaller, error) {
	kubectlVersion := extractKubectlVersionFromK3s(k3sVersion)
	if kubectlVersion == "" {
		return nil, fmt.Errorf("unable to extract kubectl version from k3s version: %s", k3sVersion)
	}

	return &ToolInstaller{
		kubectlVersion: kubectlVersion,
		helmVersion:    "", // Helm version determined by the package manager
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

// GetKubectlVersion returns the kubectl version that will be installed
func (t *ToolInstaller) GetKubectlVersion() string {
	return t.kubectlVersion
}

// GetHelmVersion returns the helm version that will be installed
func (t *ToolInstaller) GetHelmVersion() string {
	return t.helmVersion
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

// IsKubectlAIInstalled checks if kubectl-ai is available.
// On Windows it is installed as a krew plugin; the binary is named kubectl-ai
// and placed in the krew bin directory (which must be in PATH).
func (t *ToolInstaller) IsKubectlAIInstalled() bool {
	_, err := exec.LookPath("kubectl-ai")
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
func (t *ToolInstaller) InstallBrew() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("Homebrew installation is only supported on macOS")
	}

	fmt.Println("Installing Homebrew package manager...")

	// NONINTERACTIVE=1 prevents the script from prompting for user input
	cmd := exec.Command("bash", "-c",
		`/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "NONINTERACTIVE=1")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install Homebrew: %w", err)
	}

	fmt.Println("✓ Homebrew installed successfully")
	return nil
}

// IsSnapdInstalled checks if snapd is available (Linux)
func (t *ToolInstaller) IsSnapdInstalled() bool {
	_, err := exec.LookPath("snap")
	return err == nil
}

// InstallSnapd installs snapd on Debian/Ubuntu-based Linux systems.
func (t *ToolInstaller) InstallSnapd() error {
	fmt.Println("snapd not found. Installing snapd...")
	if err := t.runCommand("apt-get", "update"); err != nil {
		return fmt.Errorf("apt-get update failed: %w", err)
	}
	if err := t.runCommand("apt-get", "install", "-y", "snapd"); err != nil {
		return fmt.Errorf("apt-get install snapd failed: %w", err)
	}
	if err := t.runCommand("snap", "install", "snapd"); err != nil {
		return fmt.Errorf("snap install snapd failed: %w", err)
	}
	fmt.Println("✓ snapd installed successfully")
	return nil
}

// EnsurePackageManager detects the current OS and ensures the appropriate
// package manager is available:
//   - macOS: Homebrew (brew)
//   - Linux: snapd (snap)
//   - Windows: winget
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
	case "linux":
		if !t.IsSnapdInstalled() {
			if err := t.InstallSnapd(); err != nil {
				return fmt.Errorf("failed to install snapd: %w", err)
			}
		} else {
			fmt.Println("✓ snapd is already installed")
		}
	case "windows":
		if !t.IsWingetInstalled() {
			return fmt.Errorf("winget is not available on this system. " +
				"Please install Windows Package Manager (winget) from " +
				"https://aka.ms/getwinget and re-run this command")
		}
		fmt.Println("✓ winget is already installed")
	default:
		return fmt.Errorf(
			"automatic tool installation is only supported on macOS (Homebrew), Linux (snapd), and Windows (winget). "+
				"On %s, please install the required tools manually: hcloud, kubectl, helm, kubectl-ai",
			runtime.GOOS,
		)
	}
	return nil
}

// installWithBrew installs a package using Homebrew.
func (t *ToolInstaller) installWithBrew(formula string) error {
	fmt.Printf("Installing %s via Homebrew...\n", formula)
	if err := t.runCommand("brew", "install", formula); err != nil {
		return fmt.Errorf("brew install %s failed: %w", formula, err)
	}
	fmt.Printf("✓ %s installed successfully via Homebrew\n", formula)
	return nil
}

// installWithWinget installs a package using winget with exact ID matching.
func (t *ToolInstaller) installWithWinget(packageID string) error {
	fmt.Printf("Installing %s via winget...\n", packageID)
	err := t.runCommand("winget", "install",
		"-e",
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

// installWithSnap installs a snap package using --classic confinement.
func (t *ToolInstaller) installWithSnap(packageName string) error {
	fmt.Printf("Installing %s via snap...\n", packageName)
	if err := t.runCommand("snap", "install", packageName, "--classic"); err != nil {
		return fmt.Errorf("snap install %s failed: %w", packageName, err)
	}
	fmt.Printf("✓ %s installed successfully via snap\n", packageName)
	return nil
}

// installTool installs a named tool using the native package manager for the
// current OS. Returns an error on unsupported platforms.
func (t *ToolInstaller) installTool(toolName string) error {
	switch runtime.GOOS {
	case "darwin":
		formula, ok := brewTools[toolName]
		if !ok {
			return fmt.Errorf("no Homebrew formula defined for tool %q", toolName)
		}
		return t.installWithBrew(formula)
	case "linux":
		pkg, ok := snapTools[toolName]
		if !ok {
			return fmt.Errorf("no snap package defined for tool %q", toolName)
		}
		return t.installWithSnap(pkg)
	case "windows":
		pkgID, ok := wingetTools[toolName]
		if !ok {
			return fmt.Errorf("no winget package defined for tool %q", toolName)
		}
		return t.installWithWinget(pkgID)
	default:
		return fmt.Errorf(
			"tool installation via package manager is not supported on %s; please install %s manually",
			runtime.GOOS, toolName,
		)
	}
}

// InstallHcloud installs the hcloud CLI.
// macOS: brew install hcloud
// Linux: download and install the latest deb package from GitHub releases
// Windows: winget install -e --id HetznerCloud.CLI
func (t *ToolInstaller) InstallHcloud() error {
	fmt.Println("Installing hcloud CLI...")
	if runtime.GOOS == "linux" {
		return t.installHcloudLinux()
	}
	return t.installTool("hcloud")
}

// installHcloudLinux downloads and installs the latest hcloud deb package
// from GitHub releases, choosing the correct architecture variant.
func (t *ToolInstaller) installHcloudLinux() error {
	arch := runtime.GOARCH
	var debArch string
	switch arch {
	case "amd64":
		debArch = "amd64"
	case "arm64":
		debArch = "arm64"
	default:
		return fmt.Errorf("unsupported architecture for hcloud deb install: %s", arch)
	}

	// Resolve the latest release tag by following GitHub's /releases/latest redirect.
	// curl -L follows the redirect chain; -w %{url_effective} prints the final URL
	// which ends with /releases/tag/vX.Y.Z.
	cmd := exec.Command("curl", "-fsSLI", "-o", "/dev/null", "-w", "%{url_effective}",
		"https://github.com/hetznercloud/cli/releases/latest")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to resolve hcloud latest release URL: %w", err)
	}
	effectiveURL := strings.TrimSpace(string(out))
	// URL ends with /releases/tag/vX.Y.Z
	parts := strings.Split(effectiveURL, "/")
	if len(parts) < 2 {
		return fmt.Errorf("unexpected redirect URL for hcloud latest release: %s", effectiveURL)
	}
	tag := parts[len(parts)-1]
	if tag == "" {
		return fmt.Errorf("hcloud latest release tag is empty")
	}
	version := strings.TrimPrefix(tag, "v")

	debFile := fmt.Sprintf("hcloud-linux-%s.deb", debArch)
	downloadURL := fmt.Sprintf("https://github.com/hetznercloud/cli/releases/download/%s/%s", tag, debFile)
	tmpPath := fmt.Sprintf("/tmp/%s", debFile)

	fmt.Printf("Downloading hcloud %s (%s)...\n", tag, debArch)
	if err := t.runCommand("curl", "-fsSL", "-o", tmpPath, downloadURL); err != nil {
		return fmt.Errorf("failed to download hcloud deb package: %w", err)
	}
	defer os.Remove(tmpPath)

	fmt.Printf("Installing hcloud %s...\n", version)
	if err := t.runCommand("dpkg", "-i", tmpPath); err != nil {
		return fmt.Errorf("dpkg install of hcloud failed: %w", err)
	}

	fmt.Printf("✓ hcloud %s installed successfully\n", version)
	return nil
}

// InstallKubectl installs kubectl.
// macOS: brew install kubernetes-cli
// Linux: snap install kubectl --classic
// Windows: winget install -e --id Kubernetes.kubectl
func (t *ToolInstaller) InstallKubectl() error {
	fmt.Printf("Installing kubectl...\n")
	return t.installTool("kubectl")
}

// InstallHelm installs Helm.
// macOS: brew install helm
// Linux: snap install helm --classic
// Windows: winget install -e --id Helm.Helm
func (t *ToolInstaller) InstallHelm() error {
	fmt.Println("Installing helm...")
	return t.installTool("helm")
}

// InstallKubectlAI installs kubectl-ai.
// macOS: brew install kubectl-ai
// Linux: brew is not available; kubectl-ai must be installed manually
// Windows: winget install -e --id Kubernetes.krew, then kubectl krew install ai
func (t *ToolInstaller) InstallKubectlAI() error {
	fmt.Println("Installing kubectl-ai...")
	switch runtime.GOOS {
	case "darwin":
		return t.installWithBrew(brewTools["kubectl-ai"])
	case "linux":
		fmt.Println("⚠ kubectl-ai automatic installation is not supported on Linux via snap.")
		fmt.Println("  Please install kubectl-ai manually: https://github.com/GoogleCloudPlatform/kubectl-ai")
		return nil
	case "windows":
		fmt.Println("Installing krew via winget...")
		if err := t.installWithWinget("Kubernetes.krew"); err != nil {
			return fmt.Errorf("failed to install krew: %w", err)
		}
		fmt.Println("Installing kubectl-ai via krew...")
		if err := t.runCommand("kubectl", "krew", "install", "ai"); err != nil {
			return fmt.Errorf("failed to install kubectl-ai via krew: %w", err)
		}
		fmt.Println("✓ kubectl-ai installed successfully via krew")
		return nil
	default:
		return fmt.Errorf(
			"tool installation via package manager is not supported on %s; please install kubectl-ai manually",
			runtime.GOOS,
		)
	}
}

// EnsureToolsInstalled ensures the package manager is available and then checks
// and installs hcloud CLI, kubectl, helm, and kubectl-ai if needed.
// Tools are installed in the following order:
// 1. Package manager (brew on macOS, snapd on Linux, winget on Windows)
// 2. hcloud     - Hetzner Cloud CLI
// 3. kubectl    - Kubernetes command-line tool
// 4. helm       - Kubernetes package manager
// 5. kubectl-ai - AI-powered kubectl assistant
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
