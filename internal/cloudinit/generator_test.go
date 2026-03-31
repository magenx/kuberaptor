// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	generator := NewGenerator(&Config{
		SSHPort:  22,
		Packages: []string{},
	})

	result, err := generator.Generate()
	if err != nil {
		t.Fatalf("Failed to generate cloud-init: %v", err)
	}

	if result == "" {
		t.Fatal("Generated cloud-init is empty")
	}

	// Check for expected sections
	expectedStrings := []string{
		"#cloud-config",
		"preserve_hostname: true",
		"write_files:",
		"packages:",
		"runcmd:",
		"/etc/configure_ssh.sh",
		"/etc/systemd/system/ssh.socket.d/listen.conf",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected cloud-init to contain '%s', but it doesn't", expected)
		}
	}

	t.Logf("Generated cloud-init:\n%s", result)
}

func TestGenerateWithoutLocalFirewall(t *testing.T) {
	config := &Config{
		SSHPort:  2222,
		Packages: []string{},
	}

	generator := NewGenerator(config)
	cloudInit, err := generator.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if strings.Contains(cloudInit, "/usr/local/bin/firewall.sh") {
		t.Error("Generated cloud-init should not contain firewall.sh file")
	}

	if strings.Contains(cloudInit, "/usr/local/bin/firewall.sh setup") {
		t.Error("Generated cloud-init should not contain firewall setup command")
	}
}

func TestGenerateWithAdditionalPackages(t *testing.T) {
	config := &Config{
		SSHPort:  22,
		Packages: []string{"htop", "vim", "curl"},
	}

	generator := NewGenerator(config)
	cloudInit, err := generator.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that additional packages are included
	if !strings.Contains(cloudInit, "'htop'") {
		t.Error("Generated cloud-init doesn't contain htop package")
	}
	if !strings.Contains(cloudInit, "'vim'") {
		t.Error("Generated cloud-init doesn't contain vim package")
	}
	if !strings.Contains(cloudInit, "'curl'") {
		t.Error("Generated cloud-init doesn't contain curl package")
	}

	// Check that base packages are still present
	if !strings.Contains(cloudInit, "'fail2ban'") {
		t.Error("Generated cloud-init doesn't contain fail2ban package")
	}
	if !strings.Contains(cloudInit, "'wireguard'") {
		t.Error("Generated cloud-init doesn't contain wireguard package")
	}
}

func TestGenerateWithAdditionalPreK3sCommands(t *testing.T) {
	config := &Config{
		SSHPort:                  22,
		Packages:                 []string{},
		AdditionalPreK3sCommands: []string{"apt update", "apt upgrade -y"},
	}

	generator := NewGenerator(config)
	cloudInit, err := generator.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that pre-k3s commands are included
	if !strings.Contains(cloudInit, "apt update") {
		t.Error("Generated cloud-init doesn't contain 'apt update' command")
	}
	if !strings.Contains(cloudInit, "apt upgrade -y") {
		t.Error("Generated cloud-init doesn't contain 'apt upgrade -y' command")
	}

	// Extract the runcmd section
	runcmdIndex := strings.Index(cloudInit, "runcmd:")
	if runcmdIndex == -1 {
		t.Fatal("Could not find runcmd section")
	}
	runcmdSection := cloudInit[runcmdIndex:]

	// Verify that the commands appear in the runcmd section and in the correct order
	aptIndex := strings.Index(runcmdSection, "apt update")
	sshIndex := strings.Index(runcmdSection, "/etc/configure_ssh.sh")
	if aptIndex > sshIndex || aptIndex == -1 {
		t.Errorf("Pre-k3s commands should appear before configure_ssh.sh in runcmd section (apt@%d, ssh@%d)", aptIndex, sshIndex)
	}
}

func TestGenerateWithAdditionalPostK3sCommands(t *testing.T) {
	config := &Config{
		SSHPort:                   22,
		Packages:                  []string{},
		AdditionalPostK3sCommands: []string{"apt autoremove -y", "apt autoclean"},
	}

	generator := NewGenerator(config)
	cloudInit, err := generator.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that post-k3s commands are included
	if !strings.Contains(cloudInit, "apt autoremove -y") {
		t.Error("Generated cloud-init doesn't contain 'apt autoremove -y' command")
	}
	if !strings.Contains(cloudInit, "apt autoclean") {
		t.Error("Generated cloud-init doesn't contain 'apt autoclean' command")
	}

	// Extract the runcmd section
	runcmdIndex := strings.Index(cloudInit, "runcmd:")
	if runcmdIndex == -1 {
		t.Fatal("Could not find runcmd section")
	}
	runcmdSection := cloudInit[runcmdIndex:]

	// Verify that the commands appear after the configure_ssh command in the runcmd section
	autoremoveIndex := strings.Index(runcmdSection, "apt autoremove -y")
	sshIndex := strings.Index(runcmdSection, "/etc/configure_ssh.sh")
	if autoremoveIndex < sshIndex || autoremoveIndex == -1 {
		t.Errorf("Post-k3s commands should appear after configure_ssh.sh in runcmd section (autoremove@%d, ssh@%d)", autoremoveIndex, sshIndex)
	}
}

func TestGenerateWithMultilineCommands(t *testing.T) {
	config := &Config{
		SSHPort:  22,
		Packages: []string{},
		AdditionalPreK3sCommands: []string{
			"echo 'Line 1'\necho 'Line 2'\necho 'Line 3'",
		},
	}

	generator := NewGenerator(config)
	cloudInit, err := generator.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that multiline commands are formatted correctly
	if !strings.Contains(cloudInit, "echo 'Line 1'") {
		t.Error("Generated cloud-init doesn't contain first line of multiline command")
	}
	if !strings.Contains(cloudInit, "echo 'Line 2'") {
		t.Error("Generated cloud-init doesn't contain second line of multiline command")
	}
	if !strings.Contains(cloudInit, "echo 'Line 3'") {
		t.Error("Generated cloud-init doesn't contain third line of multiline command")
	}
}

func TestGenerateWithAllAdditionalSettings(t *testing.T) {
	config := &Config{
		SSHPort:                   22,
		Packages:                  []string{"htop", "vim"},
		AdditionalPreK3sCommands:  []string{"apt update"},
		AdditionalPostK3sCommands: []string{"apt autoremove -y"},
	}

	generator := NewGenerator(config)
	cloudInit, err := generator.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check all settings are present
	if !strings.Contains(cloudInit, "'htop'") {
		t.Error("Generated cloud-init doesn't contain htop package")
	}
	if !strings.Contains(cloudInit, "apt update") {
		t.Error("Generated cloud-init doesn't contain pre-k3s command")
	}
	if !strings.Contains(cloudInit, "apt autoremove -y") {
		t.Error("Generated cloud-init doesn't contain post-k3s command")
	}

	// Extract the runcmd section
	runcmdIndex := strings.Index(cloudInit, "runcmd:")
	if runcmdIndex == -1 {
		t.Fatal("Could not find runcmd section")
	}
	runcmdSection := cloudInit[runcmdIndex:]

	// Verify command order: pre -> mandatory -> post
	preIndex := strings.Index(runcmdSection, "apt update")
	mandatoryIndex := strings.Index(runcmdSection, "/etc/configure_ssh.sh")
	postIndex := strings.Index(runcmdSection, "apt autoremove -y")

	if preIndex > mandatoryIndex || preIndex == -1 {
		t.Errorf("Pre-k3s commands should appear before mandatory commands (pre@%d, mandatory@%d)", preIndex, mandatoryIndex)
	}
	if postIndex < mandatoryIndex || postIndex == -1 {
		t.Errorf("Post-k3s commands should appear after mandatory commands (post@%d, mandatory@%d)", postIndex, mandatoryIndex)
	}

	t.Logf("Generated cloud-init with all settings:\n%s", cloudInit)
}
