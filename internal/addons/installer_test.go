// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package addons

import (
	"testing"

	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/internal/util"
)

func TestNewInstaller(t *testing.T) {
	cfg := &config.Main{
		ClusterName: "test-cluster",
	}
	sshClient := util.NewSSHFromKeys([]byte("private"), []byte("public"))

	installer := NewInstaller(cfg, sshClient)

	if installer == nil {
		t.Fatal("NewInstaller returned nil")
	}
	if installer.Config == nil {
		t.Error("installer.Config is nil")
	}
	if installer.Config.ClusterName != "test-cluster" {
		t.Errorf("expected ClusterName 'test-cluster', got %q", installer.Config.ClusterName)
	}
	if installer.SSHClient == nil {
		t.Error("installer.SSHClient is nil")
	}
	if installer.ctx == nil {
		t.Error("installer.ctx is nil")
	}
}

func TestNewSystemUpgradeControllerInstaller(t *testing.T) {
	cfg := &config.Main{
		ClusterName:    "test-cluster",
		KubeconfigPath: "/tmp/nonexistent-kubeconfig",
	}
	sshClient := util.NewSSHFromKeys([]byte("private"), []byte("public"))

	installer := NewSystemUpgradeControllerInstaller(cfg, sshClient)

	if installer == nil {
		t.Fatal("NewSystemUpgradeControllerInstaller returned nil")
	}
	if installer.Config == nil {
		t.Error("installer.Config is nil")
	}
	if installer.Config.ClusterName != "test-cluster" {
		t.Errorf("expected ClusterName 'test-cluster', got %q", installer.Config.ClusterName)
	}
	if installer.SSHClient == nil {
		t.Error("installer.SSHClient is nil")
	}
	if installer.KubectlClient == nil {
		t.Error("installer.KubectlClient is nil")
	}
	if installer.ctx == nil {
		t.Error("installer.ctx is nil")
	}
}
