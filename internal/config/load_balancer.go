package config

// TargetPools represents target pool names for the load balancer
// Supports both string and array formats for convenience
type TargetPools []string

// UnmarshalYAML implements custom YAML unmarshaling for TargetPools
// to support both string (target_pools: "pool1") and array (target_pools: ["pool1", "pool2"]) formats
func (t *TargetPools) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try to unmarshal as a single string first
	var singleValue string
	if err := unmarshal(&singleValue); err == nil {
		*t = TargetPools{singleValue}
		return nil
	}

	// If that fails, try to unmarshal as a string array
	var arrayValue []string
	if err := unmarshal(&arrayValue); err != nil {
		return err
	}
	*t = TargetPools(arrayValue)
	return nil
}

// LoadBalancer represents global load balancer configuration for application traffic
type LoadBalancer struct {
	Enabled         bool                  `yaml:"enabled,omitempty"`
	Name            *string               `yaml:"name,omitempty"`
	Type            string                `yaml:"type,omitempty"`
	Locations       []string              `yaml:"locations,omitempty"` // Multi-location deployment support
	Algorithm       LoadBalancerAlgorithm `yaml:"algorithm,omitempty"`
	Services        []LoadBalancerService `yaml:"services,omitempty"`
	TargetPools     TargetPools           `yaml:"target_pools,omitempty"`
	UsePrivateIP    *bool                 `yaml:"use_private_ip,omitempty"`
	AttachToNetwork bool                  `yaml:"attach_to_network,omitempty"`
}

// SetDefaults sets default values for load balancer
func (lb *LoadBalancer) SetDefaults() {
	if lb.Type == "" {
		lb.Type = "lb11"
	}
	if lb.Algorithm.Type == "" {
		lb.Algorithm.Type = "round_robin"
	}
	// If no services are configured, add default HTTP and HTTPS
	if len(lb.Services) == 0 && lb.Enabled {
		lb.Services = []LoadBalancerService{
			{
				Protocol:        "tcp",
				ListenPort:      80,
				DestinationPort: 80,
				HealthCheck: &LoadBalancerHealthCheck{
					Protocol: "tcp",
					Port:     80,
					Interval: 15,
					Timeout:  10,
					Retries:  3,
				},
			},
			{
				Protocol:        "tcp",
				ListenPort:      443,
				DestinationPort: 443,
				HealthCheck: &LoadBalancerHealthCheck{
					Protocol: "tcp",
					Port:     443,
					Interval: 15,
					Timeout:  10,
					Retries:  3,
				},
			},
		}
	}
}

// LoadBalancerAlgorithm represents load balancer algorithm configuration
type LoadBalancerAlgorithm struct {
	Type string `yaml:"type,omitempty"`
}

// LoadBalancerService represents a load balancer service configuration
type LoadBalancerService struct {
	Protocol        string                   `yaml:"protocol"`
	ListenPort      int                      `yaml:"listen_port"`
	DestinationPort int                      `yaml:"destination_port"`
	ProxyProtocol   bool                     `yaml:"proxyprotocol,omitempty"`
	HealthCheck     *LoadBalancerHealthCheck `yaml:"health_check,omitempty"`
}

// LoadBalancerHealthCheck represents health check configuration for a service
type LoadBalancerHealthCheck struct {
	Protocol string                       `yaml:"protocol"`
	Port     int                          `yaml:"port"`
	Interval int                          `yaml:"interval"` // in seconds
	Timeout  int                          `yaml:"timeout"`  // in seconds
	Retries  int                          `yaml:"retries"`
	HTTP     *LoadBalancerHealthCheckHTTP `yaml:"http,omitempty"`
}

// LoadBalancerHealthCheckHTTP represents HTTP-specific health check settings
type LoadBalancerHealthCheckHTTP struct {
	Domain      string   `yaml:"domain,omitempty"`
	Path        string   `yaml:"path,omitempty"`
	StatusCodes []string `yaml:"status_codes,omitempty"`
	TLS         bool     `yaml:"tls,omitempty"`
}
