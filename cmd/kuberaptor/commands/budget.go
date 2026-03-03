// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package commands

import (
	"fmt"

	"github.com/magenx/kuberaptor/internal/cluster"
	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/pkg/hetzner"
	"github.com/spf13/cobra"
)

var (
	budgetConfigPath string
)

var budgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Show estimated monthly cost of cluster resources",
	Long:  `Display the estimated monthly cost of all cluster resources on Hetzner Cloud.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		printBanner()

		if budgetConfigPath == "" {
			return fmt.Errorf("configuration file path is required")
		}

		fmt.Printf("Loading configuration from: %s\n", budgetConfigPath)

		// Load configuration
		loader, err := config.NewLoader(budgetConfigPath, "", false)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Validate configuration for budget
		if err := loader.Validate("budget"); err != nil {
			if loader.HasErrors() {
				loader.PrintErrors()
			}
			return err
		}

		fmt.Println("\n\x1b[32mConfiguration validated successfully\x1b[0m")
		fmt.Printf("Cluster Name: %s\n\n", loader.Settings.ClusterName)

		// Create Hetzner client
		hetznerClient := hetzner.NewClient(loader.Settings.HetznerToken)

		// Create budget calculator
		calculator := cluster.NewBudgetCalculator(loader.Settings, hetznerClient)

		// Run budget calculation
		fmt.Println()
		if err := calculator.Run(); err != nil {
			return fmt.Errorf("budget calculation failed: %w", err)
		}

		return nil
	},
}

func init() {
	budgetCmd.Flags().StringVarP(&budgetConfigPath, "config", "c", "", "Path to the YAML configuration file (required)")
	budgetCmd.MarkFlagRequired("config")
}
