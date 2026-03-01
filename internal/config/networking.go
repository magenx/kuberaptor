package config

import (
	"fmt"
	"os"
)

// Default version constants
const (
	// DefaultCiliumVersion is the default Cilium version to install
	DefaultCiliumVersion = "1.17.2"
)

// Networking represents networking configuration
type Networking struct {
	CNI             CNI             `yaml:"cni,omitempty"`
	PrivateNetwork  PrivateNetwork  `yaml:"private_network,omitempty"`
	PublicNetwork   PublicNetwork   `yaml:"public_network,omitempty"`
	AllowedNetworks AllowedNetworks `yaml:"allowed_networks,omitempty"`
	SSH             SSH             `yaml:"ssh,omitempty"`
	ClusterCIDR     string          `yaml:"cluster_cidr,omitempty"`
	ServiceCIDR     string          `yaml:"service_cidr,omitempty"`
	ClusterDNS      string          `yaml:"cluster_dns,omitempty"`
}

// SetDefaults sets default values for networking
func (n *Networking) SetDefaults() {
	if n.ClusterCIDR == "" {
		n.ClusterCIDR = "10.244.0.0/16"
	}
	if n.ServiceCIDR == "" {
		n.ServiceCIDR = "10.43.0.0/16"
	}
	if n.ClusterDNS == "" {
		n.ClusterDNS = "10.43.0.10"
	}
	n.CNI.SetDefaults()
	n.PrivateNetwork.SetDefaults()
	n.PublicNetwork.SetDefaults()
	n.AllowedNetworks.SetDefaults()
	n.SSH.SetDefaults()
}

// CNI represents CNI configuration
type CNI struct {
	Enabled bool     `yaml:"enabled,omitempty"`
	Mode    string   `yaml:"mode,omitempty"`
	Cilium  *Cilium  `yaml:"cilium,omitempty"`
	Flannel *Flannel `yaml:"flannel,omitempty"`
}

// SetDefaults sets default values for CNI
func (c *CNI) SetDefaults() {
	if c.Mode == "" {
		c.Mode = "flannel"
	}
	if c.Cilium != nil {
		c.Cilium.SetDefaults()
	}
	if c.Flannel != nil {
		c.Flannel.SetDefaults()
	}
}

// Cilium represents Cilium CNI configuration
type Cilium struct {
	Enabled               bool     `yaml:"enabled,omitempty"`
	Version               string   `yaml:"version,omitempty"`         // Cilium version for CLI installation
	EncryptionType        string   `yaml:"encryption_type,omitempty"` // wireguard or ipsec
	RoutingMode           string   `yaml:"routing_mode,omitempty"`    // tunnel or native
	TunnelProtocol        string   `yaml:"tunnel_protocol,omitempty"` // vxlan or geneve
	HubbleEnabled         *bool    `yaml:"hubble_enabled,omitempty"`
	HubbleMetrics         []string `yaml:"hubble_metrics,omitempty"`
	HubbleRelayEnabled    *bool    `yaml:"hubble_relay_enabled,omitempty"`
	HubbleUIEnabled       *bool    `yaml:"hubble_ui_enabled,omitempty"`
	K8sServiceHost        string   `yaml:"k8s_service_host,omitempty"`
	K8sServicePort        int      `yaml:"k8s_service_port,omitempty"`
	OperatorReplicas      int      `yaml:"operator_replicas,omitempty"`
	OperatorMemoryRequest string   `yaml:"operator_memory_request,omitempty"`
	AgentMemoryRequest    string   `yaml:"agent_memory_request,omitempty"`
	EgressGatewayEnabled  bool     `yaml:"egress_gateway_enabled,omitempty"`
}

// SetDefaults sets default values for Cilium
func (c *Cilium) SetDefaults() {
	// Set default version
	if c.Version == "" {
		c.Version = DefaultCiliumVersion
	}
	if c.EncryptionType == "" {
		c.EncryptionType = "wireguard"
	}
	if c.RoutingMode == "" {
		c.RoutingMode = "tunnel"
	}
	if c.TunnelProtocol == "" {
		c.TunnelProtocol = "vxlan"
	}
	if c.HubbleEnabled == nil {
		defaultTrue := true
		c.HubbleEnabled = &defaultTrue
	}
	if c.HubbleRelayEnabled == nil {
		defaultTrue := true
		c.HubbleRelayEnabled = &defaultTrue
	}
	if c.HubbleUIEnabled == nil {
		defaultTrue := true
		c.HubbleUIEnabled = &defaultTrue
	}
	if c.K8sServiceHost == "" {
		c.K8sServiceHost = "127.0.0.1"
	}
	if c.K8sServicePort == 0 {
		c.K8sServicePort = 6444
	}
	if c.OperatorReplicas == 0 {
		c.OperatorReplicas = 1
	}
	if c.OperatorMemoryRequest == "" {
		c.OperatorMemoryRequest = "128Mi"
	}
	if c.AgentMemoryRequest == "" {
		c.AgentMemoryRequest = "512Mi"
	}
}

// Flannel represents Flannel CNI configuration
type Flannel struct {
	DisableKubeProxy bool  `yaml:"disable_kube_proxy,omitempty"`
	Encryption       *bool `yaml:"encryption,omitempty"`
}

// SetDefaults sets default values for Flannel
func (f *Flannel) SetDefaults() {
	// Default is false for disable_kube_proxy (kube-proxy enabled)
	// Default is true for encryption
	if f.Encryption == nil {
		defaultEncryption := true
		f.Encryption = &defaultEncryption
	}
}

// IsEncryptionEnabled returns true if encryption is enabled
func (f *Flannel) IsEncryptionEnabled() bool {
	if f.Encryption == nil {
		return true // Default is true
	}
	return *f.Encryption
}

// PrivateNetwork represents private network configuration
type PrivateNetwork struct {
	Enabled             bool        `yaml:"enabled,omitempty"`
	Subnet              string      `yaml:"subnet,omitempty"`
	ExistingNetworkName string      `yaml:"existing_network_name,omitempty"`
	NATGateway          *NATGateway `yaml:"nat_gateway,omitempty"`
}

// SetDefaults sets default values for private network
func (p *PrivateNetwork) SetDefaults() {
	if !p.Enabled {
		p.Enabled = true
	}
	if p.Subnet == "" {
		p.Subnet = "10.0.0.0/16"
	}
	if p.NATGateway != nil {
		p.NATGateway.SetDefaults()
	}
}

// NATGateway represents NAT gateway configuration for outbound internet access
type NATGateway struct {
	Enabled      bool           `yaml:"enabled,omitempty"`
	InstanceType string         `yaml:"instance_type,omitempty"`
	Locations    []string       `yaml:"locations,omitempty"` // Multi-location deployment support
	Hetzner      *HetznerConfig `yaml:"hetzner,omitempty"`   // Hetzner Cloud metadata configuration
}

// SetDefaults sets default values for NAT gateway
func (n *NATGateway) SetDefaults() {
	if n.InstanceType == "" {
		n.InstanceType = "cpx11" // Smallest instance for NAT gateway
	}
}

// HetznerLabels returns the Hetzner labels for NAT gateway
func (n *NATGateway) HetznerLabels() []Label {
	if n.Hetzner != nil {
		return n.Hetzner.Labels
	}
	return nil
}

// PublicNetwork represents public network configuration
type PublicNetwork struct {
	IPv4 *PublicNetworkIPv4 `yaml:"ipv4,omitempty"`
	IPv6 *PublicNetworkIPv6 `yaml:"ipv6,omitempty"`
}

// SetDefaults sets default values for public network
func (p *PublicNetwork) SetDefaults() {
	// Defaults will be set by validators
}

// PublicNetworkIPv4 represents IPv4 public network configuration
type PublicNetworkIPv4 struct {
	Enabled bool `yaml:"enabled,omitempty"`
}

// UnmarshalYAML implements custom YAML unmarshaling for PublicNetworkIPv4
// to support both boolean (ipv4: true) and object (ipv4: {enabled: true}) formats
func (p *PublicNetworkIPv4) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try to unmarshal as a boolean first (backward compatibility)
	var boolValue bool
	if err := unmarshal(&boolValue); err == nil {
		p.Enabled = boolValue
		return nil
	}

	// If that fails, try to unmarshal as a struct
	type rawPublicNetworkIPv4 PublicNetworkIPv4
	var raw rawPublicNetworkIPv4
	if err := unmarshal(&raw); err != nil {
		return err
	}
	*p = PublicNetworkIPv4(raw)
	return nil
}

// PublicNetworkIPv6 represents IPv6 public network configuration
type PublicNetworkIPv6 struct {
	Enabled bool `yaml:"enabled,omitempty"`
}

// UnmarshalYAML implements custom YAML unmarshaling for PublicNetworkIPv6
// to support both boolean (ipv6: true) and object (ipv6: {enabled: true}) formats
func (p *PublicNetworkIPv6) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try to unmarshal as a boolean first (backward compatibility)
	var boolValue bool
	if err := unmarshal(&boolValue); err == nil {
		p.Enabled = boolValue
		return nil
	}

	// If that fails, try to unmarshal as a struct
	type rawPublicNetworkIPv6 PublicNetworkIPv6
	var raw rawPublicNetworkIPv6
	if err := unmarshal(&raw); err != nil {
		return err
	}
	*p = PublicNetworkIPv6(raw)
	return nil
}

// AllowedNetworks represents allowed networks configuration
type AllowedNetworks struct {
	SSH []string `yaml:"ssh,omitempty"`
	API []string `yaml:"api,omitempty"`
}

// SetDefaults sets default values for allowed networks
func (a *AllowedNetworks) SetDefaults() {
	// Defaults will be set by validators
}

// SSH represents SSH configuration
type SSH struct {
	Port           int    `yaml:"port,omitempty"`
	UseAgent       bool   `yaml:"use_agent,omitempty"`
	PrivateKeyPath string `yaml:"private_key_path,omitempty"`
	PublicKeyPath  string `yaml:"public_key_path,omitempty"`
	PrivateKey     string `yaml:"private_key,omitempty"` // Inline private key content
	PublicKey      string `yaml:"public_key,omitempty"`  // Inline public key content
}

// SetDefaults sets default values for SSH
func (s *SSH) SetDefaults() {
	if s.Port == 0 {
		s.Port = 22
	}
	// Only set default paths if neither path nor inline key is provided
	if s.PrivateKeyPath == "" && s.PrivateKey == "" {
		s.PrivateKeyPath = "~/.ssh/id_rsa"
	}
	if s.PublicKeyPath == "" && s.PublicKey == "" {
		s.PublicKeyPath = "~/.ssh/id_rsa.pub"
	}
}

// ExpandedPrivateKeyPath returns the expanded private key path
func (s *SSH) ExpandedPrivateKeyPath() (string, error) {
	return ExpandPath(s.PrivateKeyPath)
}

// ExpandedPublicKeyPath returns the expanded public key path
func (s *SSH) ExpandedPublicKeyPath() (string, error) {
	return ExpandPath(s.PublicKeyPath)
}

// GetPrivateKey returns the private key content, either from inline content or file
func (s *SSH) GetPrivateKey() ([]byte, error) {
	if s.PrivateKey != "" {
		return []byte(s.PrivateKey), nil
	}
	if s.PrivateKeyPath != "" {
		expandedPath, err := s.ExpandedPrivateKeyPath()
		if err != nil {
			return nil, err
		}
		return os.ReadFile(expandedPath)
	}
	return nil, fmt.Errorf("no private key configured")
}

// GetPublicKey returns the public key content, either from inline content or file
func (s *SSH) GetPublicKey() ([]byte, error) {
	if s.PublicKey != "" {
		return []byte(s.PublicKey), nil
	}
	if s.PublicKeyPath != "" {
		expandedPath, err := s.ExpandedPublicKeyPath()
		if err != nil {
			return nil, err
		}
		return os.ReadFile(expandedPath)
	}
	return nil, fmt.Errorf("no public key configured")
}
