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
	runConfigPath string
	runCommand    string
	runScript     string
	runInstance   string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a command or script on all nodes in the cluster",
	Long:  `Run a command or script on all nodes in the cluster, or on a specific instance.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		printBanner()

		if runConfigPath == "" {
			return fmt.Errorf("configuration file path is required")
		}

		// Validate that exactly one of --command or --script is provided
		commandProvided := runCommand != ""
		scriptProvided := runScript != ""

		if commandProvided && scriptProvided {
			return fmt.Errorf("please specify either --command or --script, but not both")
		}

		if !commandProvided && !scriptProvided {
			return fmt.Errorf("please specify either --command or --script")
		}

		fmt.Printf("Loading configuration from: %s\n", runConfigPath)

		// Load configuration
		loader, err := config.NewLoader(runConfigPath, "", true)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Validate configuration for run
		if err := loader.Validate("run"); err != nil {
			if loader.HasErrors() {
				loader.PrintErrors()
			}
			return err
		}

		fmt.Println("\n\x1b[32mConfiguration validated successfully\x1b[0m")
		fmt.Printf("Cluster Name: %s\n\n", loader.Settings.ClusterName)

		// Create Hetzner client
		hetznerClient := hetzner.NewClient(loader.Settings.HetznerToken)

		// Create enhanced runner with parallel execution
		runner, err := cluster.NewRunnerEnhanced(loader.Settings, hetznerClient)
		if err != nil {
			return fmt.Errorf("failed to create runner: %w", err)
		}

		// Run command or script
		if commandProvided {
			fmt.Println("Running command with parallel execution")
			return runner.RunCommand(runCommand, runInstance)
		} else {
			fmt.Println("Running script with parallel execution")
			return runner.RunScript(runScript, runInstance)
		}
	},
}

func init() {
	runCmd.Flags().StringVarP(&runConfigPath, "config", "c", "", "Path to the YAML configuration file (required)")
	runCmd.Flags().StringVar(&runCommand, "command", "", "The command to execute on nodes")
	runCmd.Flags().StringVar(&runScript, "script", "", "The path to the script file to execute on nodes")
	runCmd.Flags().StringVar(&runInstance, "instance", "", "The instance name to run the command/script on (optional)")
	runCmd.MarkFlagRequired("config")
}
