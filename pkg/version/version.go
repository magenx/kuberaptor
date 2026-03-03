// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package version

// Version is the application version. It should be set at build time using ldflags.
// Example: go build -ldflags "-X github.com/magenx/kuberaptor/pkg/version.Version=1.0.0"
var Version = "dev"

// Get returns the current application version
func Get() string {
	return Version
}
