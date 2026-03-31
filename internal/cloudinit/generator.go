// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"
)

// Embedded templates from the templates directory
//
//go:embed templates/ssh_listen.conf
var sshListenConfTemplate string

//go:embed templates/ssh_configure.sh
var sshConfigureScript string

//go:embed templates/cloud_init.yaml
var cloudInitTemplate string

//go:embed templates/nat_gateway_cloud_init.yaml
var natGatewayCloudInitTemplate string

//go:embed templates/k3s/install_master.sh
var k3sInstallMasterTemplate string

//go:embed templates/k3s/install_worker.sh
var k3sInstallWorkerTemplate string

//go:embed templates/k3s/test_connectivity.sh
var k3sTestConnectivityTemplate string

// Config holds configuration for cloud-init generation
type Config struct {
	SSHPort                   int
	Packages                  []string
	InitCommands              []string // Shell scripts to be embedded as init files
	AdditionalPreK3sCommands  []string // Commands executed before k3s installation
	AdditionalPostK3sCommands []string // Commands executed after k3s is installed and configured
	ClusterCIDR               string
	ServiceCIDR               string
	AllowedNetworksSSH        []string
	AllowedNetworksAPI        []string
}

// Generator generates cloud-init YAML
type Generator struct {
	config *Config
}

// NewGenerator creates a new cloud-init generator
func NewGenerator(config *Config) *Generator {
	return &Generator{
		config: config,
	}
}

// Generate generates the complete cloud-init YAML
func (g *Generator) Generate() (string, error) {
	// Generate SSH files section
	sshFiles, err := g.generateSSHFiles()
	if err != nil {
		return "", fmt.Errorf("failed to generate SSH files: %w", err)
	}

	// Generate packages string
	packagesStr := g.generatePackagesStr()

	// Generate post-create commands
	postCreateCommandsStr := g.generatePostCreateCommandsStr()

	// Generate init files (script files)
	initFiles := g.generateInitFiles()

	// Render the main cloud-init template
	tmpl, err := template.New("cloud-init").Parse(cloudInitTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse cloud-init template: %w", err)
	}

	data := map[string]interface{}{
		"growpart_str":             "",
		"growroot_disabled_file":   "",
		"eth1_str":                 "",
		"firewall_files":           "",
		"ssh_files":                sshFiles,
		"init_files":               initFiles,
		"packages_str":             packagesStr,
		"post_create_commands_str": postCreateCommandsStr,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute cloud-init template: %w", err)
	}

	return buf.String(), nil
}

// generateSSHFiles generates the SSH configuration files section
func (g *Generator) generateSSHFiles() (string, error) {
	// Generate listen.conf content
	listenConfContent, err := g.renderSSHListenConf()
	if err != nil {
		return "", fmt.Errorf("failed to render listen.conf: %w", err)
	}

	// Generate configure_ssh.sh content
	configureScriptContent, err := g.renderSSHConfigureScript()
	if err != nil {
		return "", fmt.Errorf("failed to render configure_ssh.sh: %w", err)
	}

	// Encode both files
	listenConfEncoded := g.encodeAndFormat(listenConfContent)
	configureScriptEncoded := g.encodeAndFormat(configureScriptContent)

	// Build the YAML section with SSH configuration files
	sshFiles := fmt.Sprintf(`- content: %s
  path: /etc/systemd/system/ssh.socket.d/listen.conf
  encoding: gzip+base64
- content: %s
  permissions: '0755'
  path: /etc/configure_ssh.sh
  encoding: gzip+base64`, listenConfEncoded, configureScriptEncoded)

	return sshFiles, nil
}

// renderSSHListenConf renders the SSH listen.conf template
func (g *Generator) renderSSHListenConf() (string, error) {
	tmpl, err := template.New("listen.conf").Parse(sshListenConfTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]interface{}{
		"ssh_port": g.config.SSHPort,
	}); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderSSHConfigureScript renders the SSH configure script template
func (g *Generator) renderSSHConfigureScript() (string, error) {
	tmpl, err := template.New("configure_ssh.sh").Parse(sshConfigureScript)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]interface{}{
		"ssh_port": g.config.SSHPort,
	}); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// encodeAndFormat encodes content with gzip+base64 and formats it for YAML
func (g *Generator) encodeAndFormat(content string) string {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	if _, err := gzipWriter.Write([]byte(content)); err != nil {
		// This shouldn't fail in practice for in-memory writes, but handle it
		panic(fmt.Sprintf("failed to write to gzip writer: %v", err))
	}
	gzipWriter.Close()

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Format for YAML - add pipe and indent each line
	return "|\n    " + encoded
}

// generatePackagesStr generates the packages string for cloud-init
func (g *Generator) generatePackagesStr() string {
	basePackages := []string{"fail2ban", "wireguard"}
	allPackages := append(basePackages, g.config.Packages...)

	// Format as quoted strings separated by commas
	quotedPackages := make([]string, len(allPackages))
	for i, pkg := range allPackages {
		quotedPackages[i] = fmt.Sprintf("'%s'", pkg)
	}

	return strings.Join(quotedPackages, ", ")
}

// generatePostCreateCommandsStr generates the post-create commands string
func (g *Generator) generatePostCreateCommandsStr() string {
	var allCommands []string

	// Add additional pre-k3s commands first
	allCommands = append(allCommands, g.formatAdditionalCommands(g.config.AdditionalPreK3sCommands)...)

	// Add mandatory commands
	mandatoryCommands := []string{
		"hostnamectl set-hostname $(curl http://169.254.169.254/hetzner/v1/metadata/hostname)",
		"update-crypto-policies --set DEFAULT:SHA1 || true",
		"/etc/configure_ssh.sh",
		"echo \"nameserver 8.8.8.8\" > /etc/k8s-resolv.conf",
	}
	allCommands = append(allCommands, mandatoryCommands...)

	// Add init script execution commands
	for i := range g.config.InitCommands {
		allCommands = append(allCommands, fmt.Sprintf("/etc/init-%d.sh", i))
	}

	// Add additional post-k3s commands last
	allCommands = append(allCommands, g.formatAdditionalCommands(g.config.AdditionalPostK3sCommands)...)

	// Format as YAML list items
	return "- " + strings.Join(allCommands, "\n- ")
}

// formatAdditionalCommands formats additional commands, handling multiline commands
func (g *Generator) formatAdditionalCommands(commands []string) []string {
	formatted := make([]string, 0, len(commands))
	for _, cmd := range commands {
		if strings.Contains(cmd, "\n") {
			// Multiline command - format as YAML multiline string
			// This creates a single string like: "|\n  line1\n  line2\n  line3"
			lines := strings.Split(cmd, "\n")
			var builder strings.Builder
			builder.WriteString("|")
			for _, line := range lines {
				builder.WriteString("\n  ")
				builder.WriteString(line)
			}
			formatted = append(formatted, builder.String())
		} else {
			formatted = append(formatted, cmd)
		}
	}
	return formatted
}

// generateInitFiles generates the init script files section
func (g *Generator) generateInitFiles() string {
	if len(g.config.InitCommands) == 0 {
		return ""
	}

	var files []string
	for i, script := range g.config.InitCommands {
		encoded := g.encodeAndFormat(script)
		file := fmt.Sprintf(`- content: %s
  path: /etc/init-%d.sh
  encoding: gzip+base64
  permissions: '0755'`, encoded, i)
		files = append(files, file)
	}

	return strings.Join(files, "\n")
}

// GenerateNATGatewayCloudInit generates cloud-init configuration for NAT gateway
func GenerateNATGatewayCloudInit(subnet string) (string, error) {
	tmpl, err := template.New("nat_gateway_cloud_init.yaml").Parse(natGatewayCloudInitTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse NAT gateway cloud-init template: %w", err)
	}

	var buf bytes.Buffer
	data := map[string]interface{}{
		"Subnet": subnet,
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute NAT gateway cloud-init template: %w", err)
	}

	return buf.String(), nil
}

// GenerateK3sInstallMasterCommand generates k3s installation command for master nodes
// All masters use the same command with --cluster-init flag, allowing parallel installation
func GenerateK3sInstallMasterCommand(k3sVersion, k3sToken, baseArgs string) (string, error) {
	tmpl, err := template.New("install_master.sh").Parse(k3sInstallMasterTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse k3s master install template: %w", err)
	}

	var buf bytes.Buffer
	data := map[string]interface{}{
		"K3sVersion": k3sVersion,
		"K3sToken":   k3sToken,
		"BaseArgs":   baseArgs,
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute k3s master install template: %w", err)
	}

	return strings.TrimSpace(buf.String()), nil
}

// GenerateK3sInstallWorkerCommand generates k3s installation command for worker nodes
func GenerateK3sInstallWorkerCommand(k3sVersion, k3sToken, k3sURL, baseArgs string) (string, error) {
	tmpl, err := template.New("install_worker.sh").Parse(k3sInstallWorkerTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse k3s worker install template: %w", err)
	}

	var buf bytes.Buffer
	data := map[string]interface{}{
		"K3sVersion": k3sVersion,
		"K3sToken":   k3sToken,
		"K3sURL":     k3sURL,
		"BaseArgs":   baseArgs,
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute k3s worker install template: %w", err)
	}

	return strings.TrimSpace(buf.String()), nil
}

// GenerateInternetConnectivityTestCommand generates internet connectivity test command
func GenerateInternetConnectivityTestCommand() (string, error) {
	tmpl, err := template.New("test_connectivity.sh").Parse(k3sTestConnectivityTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse connectivity test template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		return "", fmt.Errorf("failed to execute connectivity test template: %w", err)
	}

	return strings.TrimSpace(buf.String()), nil
}
