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
"kubectl":    "kubernetes-cli",
"kubectl-ai": "kubectl-ai",
"cilium":     "cilium-cli",
}

// wingetTools maps logical tool names to their winget package IDs.
// kubectl-ai is handled separately via krew (see InstallKubectlAI).
var wingetTools = map[string]string{
"hcloud":  "HetznerCloud.CLI",
"helm":    "Helm.Helm",
"kubectl": "Kubernetes.kubectl",
"cilium":  "Cilium.CiliumCLI",
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
helmVersion:    "", // Helm version determined by the package manager
ciliumVersion:  "", // Cilium CLI version determined by the package manager
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

// IsKubectlAIInstalled checks if kubectl-ai is available.
// On Windows it is installed as a krew plugin; the binary is named kubectl-ai
// and placed in the krew bin directory (which must be in PATH).
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

// EnsurePackageManager detects the current OS and ensures the appropriate
// package manager (brew on macOS, winget on Windows) is available.
// Returns an error on Linux and other unsupported platforms since automatic
// tool installation requires a native package manager.
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
return fmt.Errorf(
"automatic tool installation is only supported on macOS (Homebrew) and Windows (winget). "+
"On %s, please install the required tools manually: hcloud, kubectl, helm, kubectl-ai, cilium",
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
// Windows: winget install -e --id HetznerCloud.CLI
func (t *ToolInstaller) InstallHcloud() error {
fmt.Println("Installing hcloud CLI...")
return t.installTool("hcloud")
}

// InstallKubectl installs kubectl.
// macOS: brew install kubernetes-cli
// Windows: winget install -e --id Kubernetes.kubectl
func (t *ToolInstaller) InstallKubectl() error {
fmt.Printf("Installing kubectl...\n")
return t.installTool("kubectl")
}

// InstallHelm installs Helm.
// macOS: brew install helm
// Windows: winget install -e --id Helm.Helm
func (t *ToolInstaller) InstallHelm() error {
fmt.Println("Installing helm...")
return t.installTool("helm")
}

// InstallKubectlAI installs kubectl-ai.
// macOS: brew install kubectl-ai
// Windows: winget install -e --id Kubernetes.krew, then kubectl krew install ai
func (t *ToolInstaller) InstallKubectlAI() error {
fmt.Println("Installing kubectl-ai...")
switch runtime.GOOS {
case "darwin":
return t.installWithBrew(brewTools["kubectl-ai"])
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

// InstallCilium installs the Cilium CLI.
// macOS: brew install cilium-cli
// Windows: winget install -e --id Cilium.CiliumCLI
func (t *ToolInstaller) InstallCilium() error {
fmt.Println("Installing cilium CLI...")
return t.installTool("cilium")
}

// EnsureToolsInstalled ensures the package manager is available and then checks
// and installs hcloud CLI, kubectl, helm, kubectl-ai, and cilium CLI if needed.
// Tools are installed in the following order:
// 1. Package manager (brew on macOS, winget on Windows)
// 2. hcloud     - Hetzner Cloud CLI
// 3. kubectl    - Kubernetes command-line tool
// 4. helm       - Kubernetes package manager
// 5. kubectl-ai - AI-powered kubectl assistant
// 6. cilium     - Cilium CNI CLI tool
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
