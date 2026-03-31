// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Loader handles loading and validation of configuration
type Loader struct {
	ConfigFilePath string
	NewK3sVersion  string
	Force          bool
	Settings       *Main
	Errors         []string
}

// NewLoader creates a new configuration loader
func NewLoader(configFilePath, newK3sVersion string, force bool) (*Loader, error) {
	loader := &Loader{
		ConfigFilePath: configFilePath,
		NewK3sVersion:  newK3sVersion,
		Force:          force,
		Errors:         []string{},
	}

	if err := loader.load(); err != nil {
		return nil, err
	}

	return loader, nil
}

// load reads and parses the configuration file
func (l *Loader) load() error {
	// Check if file exists
	if _, err := os.Stat(l.ConfigFilePath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", l.ConfigFilePath)
	}

	// Read file
	data, err := os.ReadFile(l.ConfigFilePath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Parse YAML
	var settings Main
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("failed to parse configuration file: %w", err)
	}

	// Set defaults
	settings.SetDefaults()

	l.Settings = &settings
	return nil
}

// Validate validates the configuration
func (l *Loader) Validate(action string) error {
	if l.Settings == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Basic validation
	if err := l.validateBasics(); err != nil {
		l.Errors = append(l.Errors, err.Error())
	}

	// Action-specific validation
	switch action {
	case "create":
		l.validateForCreate()
	case "delete":
		l.validateForDelete()
	case "upgrade":
		l.validateForUpgrade()
	case "run":
		l.validateForRun()
	case "budget":
		l.validateForBudget()
	}

	if len(l.Errors) > 0 {
		return fmt.Errorf("configuration validation failed")
	}

	return nil
}

// validateBasics performs basic validation
func (l *Loader) validateBasics() error {
	if l.Settings.HetznerToken == "" {
		return fmt.Errorf("hetzner_token is required (or set HCLOUD_TOKEN environment variable)")
	}
	if l.Settings.ClusterName == "" {
		return fmt.Errorf("cluster_name is required")
	}
	if l.Settings.KubeconfigPath == "" {
		return fmt.Errorf("kubeconfig_path is required")
	}
	if l.Settings.K3sVersion == "" {
		return fmt.Errorf("k3s_version is required")
	}
	return nil
}

// validateForCreate validates configuration for create action
func (l *Loader) validateForCreate() {
	// Comprehensive validation is performed in internal/config/validator.go
	// which includes 15+ validation rules for cluster name, k3s version,
	// SSH keys, networking, master/worker pools, and external tool checks
}

// validateForDelete validates configuration for delete action
func (l *Loader) validateForDelete() {
	// Basic validation is sufficient for delete operations
	// Only cluster name and Hetzner token are required
}

// validateForUpgrade validates configuration for upgrade action
func (l *Loader) validateForUpgrade() {
	if l.NewK3sVersion == "" {
		l.Errors = append(l.Errors, "new k3s version is required for upgrade")
	}
	// Version format validation is performed in internal/config/validator.go
}

// validateForRun validates configuration for run action
func (l *Loader) validateForRun() {
	// Basic validation is sufficient for run operations
	// Command or script validation is handled by the CLI layer
}

// validateForBudget validates configuration for budget action
func (l *Loader) validateForBudget() {
	// Basic validation is sufficient for budget operations
	// Only cluster name and Hetzner token are required to fetch resources
}

// GetErrors returns validation errors
func (l *Loader) GetErrors() []string {
	return l.Errors
}

// HasErrors returns true if there are validation errors
func (l *Loader) HasErrors() bool {
	return len(l.Errors) > 0
}

// PrintErrors prints validation errors and exits
func (l *Loader) PrintErrors() {
	if len(l.Errors) == 0 {
		return
	}

	fmt.Println("\nConfiguration Errors:")
	for _, err := range l.Errors {
		fmt.Printf("  - %s\n", err)
	}
	os.Exit(1)
}
