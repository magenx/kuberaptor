package commands

import (
	"fmt"

	"github.com/magenx/kuberaptor/internal/cluster"
	"github.com/magenx/kuberaptor/internal/util"
	"github.com/magenx/kuberaptor/pkg/hetzner"
	"github.com/spf13/cobra"
)

var (
	upgradeConfigPath    string
	upgradeNewK3sVersion string
	upgradeForce         bool
	upgradeQuiet         bool
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade a cluster to a newer version of k3s",
	Long:  `Upgrade an existing k3s cluster to a new version of k3s.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		printBanner()

		if upgradeConfigPath == "" {
			return fmt.Errorf("configuration file path is required")
		}

		if upgradeNewK3sVersion == "" {
			return fmt.Errorf("new k3s version is required")
		}

		fmt.Printf("Loading configuration from: %s\n", upgradeConfigPath)

		// Load and validate configuration
		loader, err := loadAndValidateConfig(upgradeConfigPath, upgradeNewK3sVersion, upgradeForce, "upgrade")
		if err != nil {
			return err
		}

		// Ensure required tools are installed (using the NEW k3s version for kubectl)
		fmt.Println("\nChecking for required tools (kubectl, helm, kubectl-ai)")
		installer, err := util.NewToolInstaller(upgradeNewK3sVersion)
		if err != nil {
			return fmt.Errorf("failed to initialize tool installer: %w", err)
		}

		if err := installer.EnsureToolsInstalled(); err != nil {
			return fmt.Errorf("failed to install required tools: %w", err)
		}

		fmt.Println("\n\x1b[32mConfiguration validated successfully\x1b[0m")
		fmt.Printf("Cluster Name: %s\n", loader.Settings.ClusterName)
		fmt.Printf("New k3s version: %s\n\n", upgradeNewK3sVersion)

		// Create Hetzner client
		hetznerClient := hetzner.NewClient(loader.Settings.HetznerToken)

		// Create cluster upgrader
		upgrader, err := cluster.NewUpgraderEnhanced(loader.Settings, hetznerClient, upgradeNewK3sVersion, upgradeForce)
		if err != nil {
			return fmt.Errorf("failed to create cluster upgrader: %w", err)
		}

		// Run cluster upgrade
		fmt.Println("Starting cluster upgrade")
		if err := upgrader.Run(); err != nil {
			return fmt.Errorf("cluster upgrade failed: %w", err)
		}

		if !upgradeQuiet {
			printSponsorMessage()
		}

		return nil
	},
}

func init() {
	upgradeCmd.Flags().StringVarP(&upgradeConfigPath, "config", "c", "", "Path to the YAML configuration file (required)")
	upgradeCmd.Flags().StringVar(&upgradeNewK3sVersion, "new-k3s-version", "", "The new version of k3s to upgrade to (required)")
	upgradeCmd.Flags().BoolVar(&upgradeForce, "force", false, "Force upgrade without confirmation prompts")
	upgradeCmd.Flags().BoolVarP(&upgradeQuiet, "quiet", "q", false, "Suppress the sponsor message")
	upgradeCmd.MarkFlagRequired("config")
	upgradeCmd.MarkFlagRequired("new-k3s-version")
}
