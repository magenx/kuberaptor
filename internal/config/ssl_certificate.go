// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package config

// SSLCertificate represents SSL certificate configuration for managed certificates
type SSLCertificate struct {
	Enabled  bool   `yaml:"enabled,omitempty"`
	Name     string `yaml:"name,omitempty"`     // Certificate name in Hetzner
	Domain   string `yaml:"domain,omitempty"`   // Domain for the certificate
	Preserve bool   `yaml:"preserve,omitempty"` // Preserve certificate during cluster deletion to avoid Let's Encrypt rate limits
}

// SetDefaults sets default values for SSL certificate configuration
func (s *SSLCertificate) SetDefaults() {
	// No defaults are set here - name and domain are derived from the main
	// configuration's domain field when creating the certificate if not explicitly set
}
