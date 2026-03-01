package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfigLoadingWithDNSZone(t *testing.T) {
	configYAML := `---
hetzner_token: test_token
cluster_name: demo
kubeconfig_path: "~/.kube/config"
k3s_version: v1.35.0+k3s1
domain: example.com
dns_zone:
  enabled: true
  ttl: 3600
masters_pool:
  instance_type: cx22
  instance_count: 3
  locations:
    - fsn1
load_balancer:
  enabled: true
networking:
  ssh:
    port: 22
    public_key_path: "~/.ssh/id_rsa.pub"
    private_key_path: "~/.ssh/id_rsa"
`

	var cfg Main
	err := yaml.Unmarshal([]byte(configYAML), &cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	cfg.SetDefaults()

	if cfg.Domain != "example.com" {
		t.Errorf("Expected domain 'example.com', got '%s'", cfg.Domain)
	}

	if !cfg.DNSZone.Enabled {
		t.Error("Expected DNS zone to be enabled")
	}

	if cfg.DNSZone.TTL != 3600 {
		t.Errorf("Expected DNS zone TTL 3600, got %d", cfg.DNSZone.TTL)
	}

	t.Logf("Config loaded successfully: domain=%s, dns_zone.enabled=%v, dns_zone.ttl=%d",
		cfg.Domain, cfg.DNSZone.Enabled, cfg.DNSZone.TTL)
}

func TestConfigLoadingWithoutDNSZone(t *testing.T) {
	configYAML := `---
hetzner_token: test_token
cluster_name: demo
kubeconfig_path: "~/.kube/config"
k3s_version: v1.35.0+k3s1
masters_pool:
  instance_type: cx22
  instance_count: 3
  locations:
    - fsn1
networking:
  ssh:
    port: 22
    public_key_path: "~/.ssh/id_rsa.pub"
    private_key_path: "~/.ssh/id_rsa"
`

	var cfg Main
	err := yaml.Unmarshal([]byte(configYAML), &cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	cfg.SetDefaults()

	if cfg.Domain != "" {
		t.Errorf("Expected empty domain, got '%s'", cfg.Domain)
	}

	if cfg.DNSZone.Enabled {
		t.Error("Expected DNS zone to be disabled")
	}

	// Should have default TTL even when disabled
	if cfg.DNSZone.TTL != 3600 {
		t.Errorf("Expected default DNS zone TTL 3600, got %d", cfg.DNSZone.TTL)
	}
}
