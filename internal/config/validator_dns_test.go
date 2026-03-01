package config

import (
	"strings"
	"testing"
)

// containsErrorMessage checks if any error in the slice contains the expected message
func containsErrorMessage(errors []string, expectedMsg string) bool {
	for _, err := range errors {
		if strings.Contains(err, expectedMsg) {
			return true
		}
	}
	return false
}

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		name        string
		domain      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid domain",
			domain:      "example.com",
			expectError: false,
		},
		{
			name:        "Valid subdomain",
			domain:      "subdomain.example.com",
			expectError: false,
		},
		{
			name:        "Valid multiple levels",
			domain:      "deep.subdomain.example.com",
			expectError: false,
		},
		{
			name:        "Empty domain",
			domain:      "",
			expectError: false, // Domain is optional
		},
		{
			name:        "Invalid domain - uppercase",
			domain:      "Example.Com",
			expectError: true,
			errorMsg:    "domain must be a valid DNS name",
		},
		{
			name:        "Invalid domain - underscore",
			domain:      "example_test.com",
			expectError: true,
			errorMsg:    "domain must be a valid DNS name",
		},
		{
			name:        "Invalid domain - no TLD",
			domain:      "example",
			expectError: true,
			errorMsg:    "domain must be a valid DNS name",
		},
		{
			name:        "Invalid domain - starts with dash",
			domain:      "-example.com",
			expectError: true,
			errorMsg:    "domain must be a valid DNS name",
		},
		{
			name:        "Invalid domain - ends with dash",
			domain:      "example-.com",
			expectError: true,
			errorMsg:    "domain must be a valid DNS name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Main{
				Domain: tt.domain,
			}
			validator := NewValidator(cfg)
			validator.validateDomain()

			hasError := len(validator.errors) > 0
			if hasError != tt.expectError {
				t.Errorf("Expected error: %v, got error: %v (errors: %v)", tt.expectError, hasError, validator.errors)
			}

			if tt.expectError && len(validator.errors) > 0 && tt.errorMsg != "" {
				if !containsErrorMessage(validator.errors, tt.errorMsg) {
					t.Errorf("Expected error message containing '%s', got: %v", tt.errorMsg, validator.errors)
				}
			}
		})
	}
}

func TestValidateDNSZone(t *testing.T) {
	tests := []struct {
		name         string
		domain       string
		dnsZone      DNSZone
		loadBalancer LoadBalancer
		expectError  bool
		expectWarn   bool
		errorMsg     string
		warnMsg      string
	}{
		{
			name:   "DNS zone disabled",
			domain: "example.com",
			dnsZone: DNSZone{
				Enabled: false,
			},
			expectError: false,
			expectWarn:  false,
		},
		{
			name:   "DNS zone enabled with domain",
			domain: "example.com",
			dnsZone: DNSZone{
				Enabled: true,
				TTL:     3600,
			},
			loadBalancer: LoadBalancer{
				Enabled: true,
			},
			expectError: false,
			expectWarn:  false,
		},
		{
			name:   "DNS zone enabled without domain",
			domain: "",
			dnsZone: DNSZone{
				Enabled: true,
				TTL:     3600,
			},
			expectError: true,
			expectWarn:  true, // Also warns about load balancer not enabled
			errorMsg:    "domain is required when dns_zone.enabled is true",
		},
		{
			name:   "DNS zone enabled without load balancer",
			domain: "example.com",
			dnsZone: DNSZone{
				Enabled: true,
				TTL:     3600,
			},
			loadBalancer: LoadBalancer{
				Enabled: false,
			},
			expectError: false,
			expectWarn:  true,
			warnMsg:     "dns_zone is enabled but load_balancer is not enabled",
		},
		{
			name:   "DNS zone with low TTL",
			domain: "example.com",
			dnsZone: DNSZone{
				Enabled: true,
				TTL:     30,
			},
			loadBalancer: LoadBalancer{
				Enabled: true,
			},
			expectError: true,
			errorMsg:    "dns_zone.ttl must be at least 60 seconds",
		},
		{
			name:   "DNS zone with high TTL",
			domain: "example.com",
			dnsZone: DNSZone{
				Enabled: true,
				TTL:     90000,
			},
			loadBalancer: LoadBalancer{
				Enabled: true,
			},
			expectError: false,
			expectWarn:  true,
			warnMsg:     "dns_zone.ttl is very high",
		},
		{
			name:   "DNS zone with minimum valid TTL",
			domain: "example.com",
			dnsZone: DNSZone{
				Enabled: true,
				TTL:     60,
			},
			loadBalancer: LoadBalancer{
				Enabled: true,
			},
			expectError: false,
			expectWarn:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Main{
				Domain:       tt.domain,
				DNSZone:      tt.dnsZone,
				LoadBalancer: tt.loadBalancer,
			}
			validator := NewValidator(cfg)
			validator.validateDNSZone()

			hasError := len(validator.errors) > 0
			if hasError != tt.expectError {
				t.Errorf("Expected error: %v, got error: %v (errors: %v)", tt.expectError, hasError, validator.errors)
			}

			hasWarn := len(validator.warnings) > 0
			if hasWarn != tt.expectWarn {
				t.Errorf("Expected warning: %v, got warning: %v (warnings: %v)", tt.expectWarn, hasWarn, validator.warnings)
			}

			if tt.expectError && len(validator.errors) > 0 && tt.errorMsg != "" {
				if !containsErrorMessage(validator.errors, tt.errorMsg) {
					t.Errorf("Expected error message containing '%s', got: %v", tt.errorMsg, validator.errors)
				}
			}

			if tt.expectWarn && len(validator.warnings) > 0 && tt.warnMsg != "" {
				if !containsErrorMessage(validator.warnings, tt.warnMsg) {
					t.Errorf("Expected warning message containing '%s', got: %v", tt.warnMsg, validator.warnings)
				}
			}
		})
	}
}
