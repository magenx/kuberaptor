package config

// APILoadBalancer represents the configuration for the Kubernetes API load balancer
type APILoadBalancer struct {
	Enabled bool           `yaml:"enabled,omitempty"`
	Hetzner *HetznerConfig `yaml:"hetzner,omitempty"` // Hetzner Cloud metadata configuration
}

// SetDefaults sets default values for API load balancer
func (a *APILoadBalancer) SetDefaults() {
	// No defaults to set currently
}
