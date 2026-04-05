// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package config

// DNSZone represents DNS zone configuration for domain management
type DNSZone struct {
	Enabled  bool   `yaml:"enabled,omitempty"`
	Name     string `yaml:"name,omitempty"`    // Optional override for zone name, defaults to domain
	TTL      int    `yaml:"ttl,omitempty"`     // TTL for DNS records in seconds
	Preserve bool   `yaml:"preserve,omitempty"` // When true, the DNS zone is not deleted on cluster deletion and is reused when a new cluster with the same zone name is created
}

// SetDefaults sets default values for DNS zone configuration
func (d *DNSZone) SetDefaults() {
	if d.TTL == 0 {
		d.TTL = 3600 // Default TTL of 1 hour
	}
}
