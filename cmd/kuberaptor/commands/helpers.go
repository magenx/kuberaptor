package commands

import (
	"fmt"

	"github.com/magenx/kuberaptor/internal/config"
)

// loadAndValidateConfig loads and validates configuration for a specific operation
// Returns the loader and any error encountered during loading or validation
func loadAndValidateConfig(configPath, newK3sVersion string, force bool, operation string) (*config.Loader, error) {
	// Load configuration
	loader, err := config.NewLoader(configPath, newK3sVersion, force)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration for the specified operation
	if err := loader.Validate(operation); err != nil {
		if loader.HasErrors() {
			loader.PrintErrors()
		}
		return nil, err
	}

	return loader, nil
}
