// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"strings"
	"testing"

	"github.com/magenx/kuberaptor/internal/config"
)

// TestConfigMergingLogic tests that pool-specific settings are appended to root settings
func TestConfigMergingLogic(t *testing.T) {
	// Root config
	rootConfig := &config.Main{
		AdditionalPackages:        []string{"htop", "vim"},
		AdditionalPreK3sCommands:  []string{"apt update"},
		AdditionalPostK3sCommands: []string{"apt autoremove -y"},
	}

	// Pool with additions
	poolWithAdditions := &config.NodePool{
		AdditionalPackages:        []string{"curl"},
		AdditionalPreK3sCommands:  []string{"echo 'pool pre'"},
		AdditionalPostK3sCommands: []string{"echo 'pool post'"},
	}

	// Pool without additions (should use root only)
	poolWithoutAdditions := &config.NodePool{
		AdditionalPackages:        nil,
		AdditionalPreK3sCommands:  nil,
		AdditionalPostK3sCommands: nil,
	}

	t.Run("pool with additions - global first, then pool", func(t *testing.T) {
		// Additive behavior: global + pool
		packages := append([]string{}, rootConfig.AdditionalPackages...)
		if len(poolWithAdditions.AdditionalPackages) > 0 {
			packages = append(packages, poolWithAdditions.AdditionalPackages...)
		}

		preCommands := append([]string{}, rootConfig.AdditionalPreK3sCommands...)
		if len(poolWithAdditions.AdditionalPreK3sCommands) > 0 {
			preCommands = append(preCommands, poolWithAdditions.AdditionalPreK3sCommands...)
		}

		postCommands := append([]string{}, rootConfig.AdditionalPostK3sCommands...)
		if len(poolWithAdditions.AdditionalPostK3sCommands) > 0 {
			postCommands = append(postCommands, poolWithAdditions.AdditionalPostK3sCommands...)
		}

		// Verify global + pool settings are combined
		if len(packages) != 3 || packages[0] != "htop" || packages[1] != "vim" || packages[2] != "curl" {
			t.Errorf("Expected packages [htop vim curl], got %v", packages)
		}
		if len(preCommands) != 2 || preCommands[0] != "apt update" || preCommands[1] != "echo 'pool pre'" {
			t.Errorf("Expected pre commands [apt update, echo 'pool pre'], got %v", preCommands)
		}
		if len(postCommands) != 2 || postCommands[0] != "apt autoremove -y" || postCommands[1] != "echo 'pool post'" {
			t.Errorf("Expected post commands [apt autoremove -y, echo 'pool post'], got %v", postCommands)
		}

		// Generate cloud-init with combined settings
		cfg := &Config{
			SSHPort:                   22,
			Packages:                  packages,
			AdditionalPreK3sCommands:  preCommands,
			AdditionalPostK3sCommands: postCommands,
		}
		generator := NewGenerator(cfg)
		cloudInit, err := generator.Generate()
		if err != nil {
			t.Fatalf("Failed to generate cloud-init: %v", err)
		}

		// Verify both global and pool-specific settings are in the cloud-init
		if !strings.Contains(cloudInit, "'htop'") {
			t.Error("Cloud-init should contain global package 'htop'")
		}
		if !strings.Contains(cloudInit, "'vim'") {
			t.Error("Cloud-init should contain global package 'vim'")
		}
		if !strings.Contains(cloudInit, "'curl'") {
			t.Error("Cloud-init should contain pool-specific package 'curl'")
		}
		if !strings.Contains(cloudInit, "apt update") {
			t.Error("Cloud-init should contain global pre command 'apt update'")
		}
		if !strings.Contains(cloudInit, "echo 'pool pre'") {
			t.Error("Cloud-init should contain pool-specific pre command")
		}
		if !strings.Contains(cloudInit, "apt autoremove -y") {
			t.Error("Cloud-init should contain global post command 'apt autoremove -y'")
		}
		if !strings.Contains(cloudInit, "echo 'pool post'") {
			t.Error("Cloud-init should contain pool-specific post command")
		}
	})

	t.Run("pool without additions uses root settings only", func(t *testing.T) {
		// Only global settings when pool has no additions
		packages := append([]string{}, rootConfig.AdditionalPackages...)
		if len(poolWithoutAdditions.AdditionalPackages) > 0 {
			packages = append(packages, poolWithoutAdditions.AdditionalPackages...)
		}

		preCommands := append([]string{}, rootConfig.AdditionalPreK3sCommands...)
		if len(poolWithoutAdditions.AdditionalPreK3sCommands) > 0 {
			preCommands = append(preCommands, poolWithoutAdditions.AdditionalPreK3sCommands...)
		}

		postCommands := append([]string{}, rootConfig.AdditionalPostK3sCommands...)
		if len(poolWithoutAdditions.AdditionalPostK3sCommands) > 0 {
			postCommands = append(postCommands, poolWithoutAdditions.AdditionalPostK3sCommands...)
		}

		// Verify only root settings are used
		if len(packages) != 2 || packages[0] != "htop" || packages[1] != "vim" {
			t.Errorf("Expected root packages [htop vim], got %v", packages)
		}
		if len(preCommands) != 1 || preCommands[0] != "apt update" {
			t.Errorf("Expected root pre commands [apt update], got %v", preCommands)
		}
		if len(postCommands) != 1 || postCommands[0] != "apt autoremove -y" {
			t.Errorf("Expected root post commands [apt autoremove -y], got %v", postCommands)
		}

		// Generate cloud-init with root settings
		cfg := &Config{
			SSHPort:                   22,
			Packages:                  packages,
			AdditionalPreK3sCommands:  preCommands,
			AdditionalPostK3sCommands: postCommands,
		}
		generator := NewGenerator(cfg)
		cloudInit, err := generator.Generate()
		if err != nil {
			t.Fatalf("Failed to generate cloud-init: %v", err)
		}

		// Verify root settings are in the cloud-init
		if !strings.Contains(cloudInit, "'htop'") {
			t.Error("Cloud-init should contain root package 'htop'")
		}
		if !strings.Contains(cloudInit, "'vim'") {
			t.Error("Cloud-init should contain root package 'vim'")
		}
		if !strings.Contains(cloudInit, "apt update") {
			t.Error("Cloud-init should contain root pre command 'apt update'")
		}
		if !strings.Contains(cloudInit, "apt autoremove -y") {
			t.Error("Cloud-init should contain root post command 'apt autoremove -y'")
		}
	})
}
