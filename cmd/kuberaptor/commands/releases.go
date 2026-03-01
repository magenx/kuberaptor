package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/magenx/kuberaptor/pkg/k3s"
	"github.com/spf13/cobra"
)

var releasesCmd = &cobra.Command{
	Use:   "releases",
	Short: "List available k3s releases",
	Long:  `List all available k3s releases from the k3s GitHub repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		printBanner()

		fmt.Println("Fetching available k3s releases")

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		releases, err := k3s.GetAvailableReleases(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch releases: %w", err)
		}

		fmt.Printf("\nFound %d releases:\n\n", len(releases))
		for _, release := range releases {
			fmt.Println(release)
		}

		return nil
	},
}
