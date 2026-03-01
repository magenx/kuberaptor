package config

import (
	"strings"
	"testing"
)

func TestValidateSSLCertificate(t *testing.T) {
	tests := []struct {
		name          string
		config        Main
		expectError   bool
		errorContains string
		warnContains  string
	}{
		{
			name: "SSL certificate disabled - no validation",
			config: Main{
				ClusterName: "test",
				SSLCertificate: SSLCertificate{
					Enabled: false,
				},
			},
			expectError: false,
		},
		{
			name: "SSL certificate enabled without DNS zone",
			config: Main{
				ClusterName: "test",
				Domain:      "example.com",
				SSLCertificate: SSLCertificate{
					Enabled: true,
				},
				DNSZone: DNSZone{
					Enabled: false,
				},
				LoadBalancer: LoadBalancer{
					Enabled: true,
					Services: []LoadBalancerService{
						{Protocol: "https", ListenPort: 443, DestinationPort: 80},
					},
				},
			},
			expectError:   true,
			errorContains: "dns_zone.enabled must be true",
		},
		{
			name: "SSL certificate enabled without domain",
			config: Main{
				ClusterName: "test",
				SSLCertificate: SSLCertificate{
					Enabled: true,
				},
				DNSZone: DNSZone{
					Enabled: true,
				},
				LoadBalancer: LoadBalancer{
					Enabled: true,
					Services: []LoadBalancerService{
						{Protocol: "https", ListenPort: 443, DestinationPort: 80},
					},
				},
			},
			expectError:   true,
			errorContains: "domain is required",
		},
		{
			name: "SSL certificate enabled without load balancer",
			config: Main{
				ClusterName: "test",
				Domain:      "example.com",
				SSLCertificate: SSLCertificate{
					Enabled: true,
				},
				DNSZone: DNSZone{
					Enabled: true,
				},
				LoadBalancer: LoadBalancer{
					Enabled: false,
				},
			},
			expectError:   true,
			errorContains: "load_balancer.enabled must be true",
		},
		{
			name: "SSL certificate enabled without HTTPS service - warning",
			config: Main{
				ClusterName: "test",
				Domain:      "example.com",
				SSLCertificate: SSLCertificate{
					Enabled: true,
				},
				DNSZone: DNSZone{
					Enabled: true,
					TTL:     3600,
				},
				LoadBalancer: LoadBalancer{
					Enabled: true,
					Services: []LoadBalancerService{
						{Protocol: "tcp", ListenPort: 80, DestinationPort: 80},
					},
				},
				Networking: Networking{
					SSH: SSH{
						PublicKeyPath:  "/tmp/test.pub",
						PrivateKeyPath: "/tmp/test",
					},
				},
			},
			expectError:  false,
			warnContains: "no HTTPS service found",
		},
		{
			name: "Valid SSL certificate configuration",
			config: Main{
				ClusterName: "test",
				Domain:      "example.com",
				SSLCertificate: SSLCertificate{
					Enabled: true,
					Name:    "example.com",
					Domain:  "example.com",
				},
				DNSZone: DNSZone{
					Enabled: true,
					TTL:     3600,
				},
				LoadBalancer: LoadBalancer{
					Enabled: true,
					Services: []LoadBalancerService{
						{Protocol: "https", ListenPort: 443, DestinationPort: 80},
					},
				},
				Networking: Networking{
					SSH: SSH{
						PublicKeyPath:  "/tmp/test.pub",
						PrivateKeyPath: "/tmp/test",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator(&tt.config)
			validator.validateSSLCertificate()

			hasError := len(validator.errors) > 0
			if hasError != tt.expectError {
				t.Errorf("Expected error: %v, got: %v (errors: %v)", tt.expectError, hasError, validator.errors)
			}

			if tt.expectError && tt.errorContains != "" {
				found := false
				for _, err := range validator.errors {
					if strings.Contains(err, tt.errorContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing '%s', got errors: %v", tt.errorContains, validator.errors)
				}
			}

			if tt.warnContains != "" {
				found := false
				for _, warn := range validator.warnings {
					if strings.Contains(warn, tt.warnContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected warning containing '%s', got warnings: %v", tt.warnContains, validator.warnings)
				}
			}
		})
	}
}
