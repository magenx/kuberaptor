// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package commands

import (
	"fmt"

	"github.com/magenx/kuberaptor/internal/cluster"
	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/internal/util"
	"github.com/magenx/kuberaptor/pkg/hetzner"
	"github.com/spf13/cobra"
)

var (
	createConfigPath string
	createQuiet      bool
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a cluster",
	Long:  `Create a new k3s cluster on Hetzner Cloud using the provided configuration file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		printBanner()

		if createConfigPath == "" {
			return fmt.Errorf("configuration file path is required")
		}

		fmt.Printf("Loading configuration from: %s\n", createConfigPath)

		// Load and validate configuration
		loader, err := loadAndValidateConfig(createConfigPath, "", true, "create")
		if err != nil {
			return err
		}

		// Ensure required tools are installed (using k3s version from config)
		fmt.Println("\nChecking for required tools:")
		installer, err := util.NewToolInstaller(loader.Settings.K3sVersion)
		if err != nil {
			return fmt.Errorf("failed to initialize tool installer: %w", err)
		}

		if err := installer.EnsureToolsInstalled(); err != nil {
			return fmt.Errorf("failed to install required tools: %w", err)
		}

		// Run comprehensive validator
		validator := config.NewValidator(loader.Settings)
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		fmt.Println("\n\033[32mConfiguration validated successfully\033[0m")
		fmt.Printf("Cluster Name: %s\n", loader.Settings.ClusterName)
		fmt.Printf("K3s Version: %s\n", loader.Settings.K3sVersion)
		fmt.Printf("Masters: %d nodes\n", loader.Settings.MastersPool.InstanceCount)
		fmt.Printf("Workers: %d pools\n\n", len(loader.Settings.WorkerNodePools))

		// Create Hetzner client
		hetznerClient := hetzner.NewClient(loader.Settings.HetznerToken)

		// Create cluster creator
		creator, err := cluster.NewCreatorEnhanced(loader.Settings, hetznerClient)
		if err != nil {
			return fmt.Errorf("failed to create cluster creator: %w", err)
		}

		// Run cluster creation
		if err := creator.Run(); err != nil {
			return fmt.Errorf("cluster creation failed: %w", err)
		}

		if !createQuiet {
			printSponsorMessage()
		}

		return nil
	},
}

func init() {
	createCmd.Flags().StringVarP(&createConfigPath, "config", "c", "", "Path to the YAML configuration file (required)")
	createCmd.Flags().BoolVarP(&createQuiet, "quiet", "q", false, "Suppress the sponsor message")
	createCmd.MarkFlagRequired("config")
}
