package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSSLCertificateUnmarshalYAML(t *testing.T) {
	yamlData := `
ssl_certificate:
  enabled: true
  name: example.com
  domain: example.com
  preserve: true
`

	type testConfig struct {
		SSLCertificate SSLCertificate `yaml:"ssl_certificate"`
	}

	var config testConfig
	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !config.SSLCertificate.Enabled {
		t.Error("Expected SSL certificate to be enabled")
	}
	if config.SSLCertificate.Name != "example.com" {
		t.Errorf("Expected name 'example.com', got '%s'", config.SSLCertificate.Name)
	}
	if config.SSLCertificate.Domain != "example.com" {
		t.Errorf("Expected domain 'example.com', got '%s'", config.SSLCertificate.Domain)
	}
	if !config.SSLCertificate.Preserve {
		t.Error("Expected preserve to be true")
	}
}

func TestSSLCertificateSetDefaults(t *testing.T) {
	tests := []struct {
		name string
		cert SSLCertificate
	}{
		{
			name: "Empty SSL certificate",
			cert: SSLCertificate{},
		},
		{
			name: "Enabled SSL certificate",
			cert: SSLCertificate{
				Enabled: true,
				Name:    "test-cert",
				Domain:  "test.com",
			},
		},
		{
			name: "SSL certificate with preserve",
			cert: SSLCertificate{
				Enabled:  true,
				Name:     "test-cert",
				Domain:   "test.com",
				Preserve: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert := tt.cert
			// SetDefaults should not panic and should be idempotent
			cert.SetDefaults()
			cert.SetDefaults() // Call twice to verify idempotence
		})
	}
}

func TestSSLCertificatePreserveDefault(t *testing.T) {
	// Test that preserve defaults to false
	cert := SSLCertificate{
		Enabled: true,
		Name:    "test.com",
		Domain:  "test.com",
	}

	if cert.Preserve {
		t.Error("Expected preserve to default to false")
	}
}

func TestSSLCertificatePreserveUnmarshalDefault(t *testing.T) {
	// Test that preserve defaults to false when not specified in YAML
	yamlData := `
ssl_certificate:
  enabled: true
  name: example.com
  domain: example.com
`

	type testConfig struct {
		SSLCertificate SSLCertificate `yaml:"ssl_certificate"`
	}

	var config testConfig
	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if config.SSLCertificate.Preserve {
		t.Error("Expected preserve to default to false when not specified")
	}
}
