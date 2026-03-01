package commands

import (
	"fmt"
	"os"

	"github.com/magenx/kuberaptor/internal/config"
	"github.com/spf13/cobra"
)

var (
	configGenerate   bool
	configOutputPath string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration utilities",
	Long:  `Generate or validate configuration files for kuberaptor clusters.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if configGenerate {
			return generateConfig()
		}
		return cmd.Help()
	},
}

func init() {
	configCmd.Flags().BoolVarP(&configGenerate, "generate", "g", false, "Generate a configuration file skeleton")
	configCmd.Flags().StringVarP(&configOutputPath, "output", "o", "cluster.yaml.generated", "Output file path for generated configuration")
}

func generateConfig() error {
	printBanner()

	fmt.Println("Generating configuration skeleton...")

	// Generate the skeleton
	data, err := config.GenerateSkeleton()
	if err != nil {
		return fmt.Errorf("failed to generate configuration skeleton: %w", err)
	}

	// Write to file with restrictive permissions (only owner can read/write)
	if err := os.WriteFile(configOutputPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	fmt.Printf("\n\033[32m✓\033[0m Configuration skeleton generated successfully!\n")
	fmt.Printf("  Output file: %s\n", configOutputPath)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Edit the generated file and fill in your desired values\n")
	fmt.Printf("  2. Create your cluster: kuberaptor create --config %s\n\n", configOutputPath)

	return nil
}
