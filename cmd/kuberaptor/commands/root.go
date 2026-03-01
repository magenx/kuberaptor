package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version string

// Execute executes the root command
func Execute(ver string) error {
	version = ver
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "kuberaptor",
	Short: "A tool to create k3s clusters on Hetzner Cloud",
	Long:  `kuberaptor - Production-ready Kubernetes clusters created with a single command. No programming required, no complexity.`,
	Run: func(cmd *cobra.Command, args []string) {
		printBanner()
		cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(releasesCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(budgetCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)

	// Disable auto-generated completion command (we have our own)
	rootCmd.CompletionOptions.DisableDefaultCmd = false
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		printBanner()
		fmt.Printf("Version: %s\n", version)
	},
}

func printBanner() {
	green := "\033[32m"
	blue := "\033[34m"
	reset := "\033[0m"

	if os.Getenv("NO_COLOR") != "" {
		green = ""
		blue = ""
		reset = ""
	}

	fmt.Printf("%s _          _                          _             %s\n", green, reset)
	fmt.Printf("%s| | ___   _| |__   ___ _ __ __ _ _ __ | |_ ___  _ __ %s\n", green, reset)
	fmt.Printf("%s| |/ / | | | '_ \\ / _ \\ '__/ _` | '_ \\| __/ _ \\| '__|%s\n", green, reset)
	fmt.Printf("%s|   <| |_| | |_) |  __/ | | (_| | |_) | || (_) | |   %s\n", green, reset)
	fmt.Printf("%s|_|\\_\\\\__,_|_.__/ \\___|_|  \\__,_| .__/ \\__\\___/|_|   %s\n", green, reset)
	fmt.Printf("%s                                 |_|                  %s\n", green, reset)
	fmt.Println()
	fmt.Printf("%sVersion: %s%s\n", blue, version, reset)
	fmt.Println()
}

func printSponsorMessage() {
	blue := "\033[34m"
	reset := "\033[0m"

	if os.Getenv("NO_COLOR") != "" {
		blue = ""
		reset = ""
	}

	fmt.Println()
	fmt.Printf("%s=======================================================%s\n", blue, reset)
	fmt.Printf("%s  Do you like kuberaptor? Support its development:%s\n", blue, reset)
	fmt.Printf("%s  https://github.com/magenx/kuberaptor%s\n", blue, reset)
	fmt.Printf("%s=======================================================%s\n", blue, reset)
	fmt.Println()
}
