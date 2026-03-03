// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package commands

import (
	"fmt"

	"github.com/magenx/kuberaptor/internal/cluster"
	"github.com/magenx/kuberaptor/pkg/hetzner"
	"github.com/spf13/cobra"
)

var (
	deleteConfigPath string
	deleteForce      bool
	deleteQuiet      bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a cluster",
	Long:  `Delete an existing k3s cluster on Hetzner Cloud.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		printBanner()

		if deleteConfigPath == "" {
			return fmt.Errorf("configuration file path is required")
		}

		fmt.Printf("Loading configuration from: %s\n", deleteConfigPath)

		// Load and validate configuration
		loader, err := loadAndValidateConfig(deleteConfigPath, "", deleteForce, "delete")
		if err != nil {
			return err
		}

		fmt.Println("\n\x1b[32mConfiguration validated successfully\x1b[0m")
		fmt.Printf("Cluster Name: %s\n\n", loader.Settings.ClusterName)

		// Create Hetzner client
		hetznerClient := hetzner.NewClient(loader.Settings.HetznerToken)

		// Create cluster deleter
		deleter := cluster.NewDeleter(loader.Settings, hetznerClient, deleteForce)

		// Run cluster deletion
		fmt.Println()
		if err := deleter.Run(); err != nil {
			return fmt.Errorf("cluster deletion failed: %w", err)
		}

		if !deleteQuiet {
			printSponsorMessage()
		}

		return nil
	},
}

func init() {
	deleteCmd.Flags().StringVarP(&deleteConfigPath, "config", "c", "", "Path to the YAML configuration file (required)")
	deleteCmd.Flags().BoolVar(&deleteForce, "force", false, "Force deletion without confirmation prompts")
	deleteCmd.Flags().BoolVarP(&deleteQuiet, "quiet", "q", false, "Suppress the sponsor message")
	deleteCmd.MarkFlagRequired("config")
}
