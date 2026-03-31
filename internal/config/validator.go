// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
)

// Validator provides configuration validation
type Validator struct {
	config   *Main
	errors   []string
	warnings []string
}

// NewValidator creates a new configuration validator
func NewValidator(config *Main) *Validator {
	return &Validator{
		config:   config,
		errors:   []string{},
		warnings: []string{},
	}
}

// Validate performs comprehensive validation
func (v *Validator) Validate() error {
	v.validateClusterName()
	v.validateK3sVersion()
	v.validateDomain()
	v.validateSSHKeys()
	v.validateNetworking()
	v.validateMasterPool()
	v.validateWorkerPools()
	v.validateDatastore()
	v.validateLoadBalancer()
	v.validateDNSZone()
	v.validateSSLCertificate()
	v.validateExternalTools()

	if len(v.errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s",
			strings.Join(v.errors, "\n  - "))
	}

	if len(v.warnings) > 0 {
		fmt.Printf("\n  Configuration warnings:\n  - %s\n\n",
			strings.Join(v.warnings, "\n  - "))
	}

	return nil
}

// validateClusterName validates cluster name format
func (v *Validator) validateClusterName() {
	if v.config.ClusterName == "" {
		v.errors = append(v.errors, "cluster_name is required")
		return
	}

	// Cluster name should be valid for DNS
	validName := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !validName.MatchString(v.config.ClusterName) {
		v.errors = append(v.errors,
			"cluster_name must start and end with alphanumeric, contain only lowercase letters, numbers, and hyphens")
	}

	if len(v.config.ClusterName) > 63 {
		v.errors = append(v.errors, "cluster_name must be 63 characters or less")
	}
}

// validateK3sVersion validates k3s version format
func (v *Validator) validateK3sVersion() {
	if v.config.K3sVersion == "" {
		v.warnings = append(v.warnings, "k3s_version not specified, will use latest stable")
		return
	}

	// K3s version should match pattern vX.Y.Z+k3sN
	validVersion := regexp.MustCompile(`^v\d+\.\d+\.\d+\+k3s\d+$`)
	if !validVersion.MatchString(v.config.K3sVersion) {
		v.errors = append(v.errors,
			"k3s_version must match format vX.Y.Z+k3sN (e.g., v1.32.0+k3s1)")
	}
}

// validateSSHKeys validates SSH key paths
func (v *Validator) validateSSHKeys() {
	// Validate public key (either path or inline content)
	if v.config.Networking.SSH.PublicKeyPath == "" && v.config.Networking.SSH.PublicKey == "" {
		v.errors = append(v.errors, "SSH public_key_path or public_key is required")
	} else if v.config.Networking.SSH.PublicKeyPath != "" && v.config.Networking.SSH.PublicKey != "" {
		v.errors = append(v.errors, "Cannot specify both SSH public_key_path and public_key - choose one")
	} else if v.config.Networking.SSH.PublicKeyPath != "" {
		// Validate path exists
		expandedPath, err := v.config.Networking.SSH.ExpandedPublicKeyPath()
		if err != nil {
			v.errors = append(v.errors,
				fmt.Sprintf("SSH public key path expansion failed: %s", err))
		} else if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			v.errors = append(v.errors,
				fmt.Sprintf("SSH public key not found: %s", v.config.Networking.SSH.PublicKeyPath))
		}
	} else if v.config.Networking.SSH.PublicKey != "" {
		// Validate inline key format (basic check)
		if len(v.config.Networking.SSH.PublicKey) < 50 {
			v.errors = append(v.errors, "SSH public_key appears to be too short to be a valid key (minimum 50 characters)")
		}
	}

	// Validate private key (either path or inline content)
	if v.config.Networking.SSH.PrivateKeyPath == "" && v.config.Networking.SSH.PrivateKey == "" {
		v.errors = append(v.errors, "SSH private_key_path or private_key is required")
	} else if v.config.Networking.SSH.PrivateKeyPath != "" && v.config.Networking.SSH.PrivateKey != "" {
		v.errors = append(v.errors, "Cannot specify both SSH private_key_path and private_key - choose one")
	} else if v.config.Networking.SSH.PrivateKeyPath != "" {
		// Validate path exists
		expandedPath, err := v.config.Networking.SSH.ExpandedPrivateKeyPath()
		if err != nil {
			v.errors = append(v.errors,
				fmt.Sprintf("SSH private key path expansion failed: %s", err))
		} else if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			v.errors = append(v.errors,
				fmt.Sprintf("SSH private key not found: %s", v.config.Networking.SSH.PrivateKeyPath))
		}
	} else if v.config.Networking.SSH.PrivateKey != "" {
		// Validate inline key format (basic check)
		if len(v.config.Networking.SSH.PrivateKey) < 100 {
			v.errors = append(v.errors, "SSH private_key appears to be too short to be a valid key (minimum 100 characters)")
		}
	}

	// Validate SSH port
	if v.config.Networking.SSH.Port < 1 || v.config.Networking.SSH.Port > 65535 {
		v.errors = append(v.errors, "SSH port must be between 1 and 65535")
	}
}

// validateNetworking validates network configuration
func (v *Validator) validateNetworking() {
	if v.config.Networking.PrivateNetwork.Enabled {
		subnet := v.config.Networking.PrivateNetwork.Subnet
		if subnet == "" {
			v.errors = append(v.errors, "private network subnet is required when private network is enabled")
		} else {
			// Validate CIDR notation
			_, _, err := net.ParseCIDR(subnet)
			if err != nil {
				v.errors = append(v.errors,
					fmt.Sprintf("invalid private network subnet CIDR: %s", subnet))
			}
		}
	}

	// Validate allowed networks
	for _, cidr := range v.config.Networking.AllowedNetworks.SSH {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			v.errors = append(v.errors,
				fmt.Sprintf("invalid SSH allowed network CIDR: %s", cidr))
		}
	}

	for _, cidr := range v.config.Networking.AllowedNetworks.API {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			v.errors = append(v.errors,
				fmt.Sprintf("invalid API allowed network CIDR: %s", cidr))
		}
	}

	// Warn about open API access
	for _, cidr := range v.config.Networking.AllowedNetworks.API {
		if cidr == "0.0.0.0/0" {
			v.warnings = append(v.warnings,
				"Kubernetes API is open to the internet (0.0.0.0/0). Consider restricting access.")
			break
		}
	}
}

// validateMasterPool validates master node pool configuration
func (v *Validator) validateMasterPool() {
	if v.config.MastersPool.InstanceType == "" {
		v.errors = append(v.errors, "master instance_type is required")
	}

	if v.config.MastersPool.InstanceCount < 1 {
		v.errors = append(v.errors, "master instance_count must be at least 1")
	}

	// Warn about even number of masters (not recommended for etcd quorum)
	if v.config.MastersPool.InstanceCount > 1 && v.config.MastersPool.InstanceCount%2 == 0 {
		v.warnings = append(v.warnings,
			fmt.Sprintf("master instance_count is %d (even). For HA, odd numbers (3, 5, 7) are recommended for etcd quorum",
				v.config.MastersPool.InstanceCount))
	}

	if len(v.config.MastersPool.Locations) == 0 {
		v.errors = append(v.errors, "at least one master location is required")
	}

	// Validate placement group if configured
	v.validatePlacementGroup(v.config.MastersPool.PlacementGroup, "master pool")
}

// validateWorkerPools validates worker node pool configurations
func (v *Validator) validateWorkerPools() {
	if len(v.config.WorkerNodePools) == 0 {
		v.warnings = append(v.warnings, "no worker pools configured, cluster will only have master nodes")
		return
	}

	poolNames := make(map[string]bool)
	for i, pool := range v.config.WorkerNodePools {
		// Check if name is provided
		if pool.Name == nil || *pool.Name == "" {
			v.errors = append(v.errors, fmt.Sprintf("worker pool %d: name is required", i))
		} else {
			if poolNames[*pool.Name] {
				v.errors = append(v.errors, fmt.Sprintf("duplicate worker pool name: %s", *pool.Name))
			}
			poolNames[*pool.Name] = true
		}

		if pool.InstanceType == "" {
			poolName := "unknown"
			if pool.Name != nil {
				poolName = *pool.Name
			}
			v.errors = append(v.errors, fmt.Sprintf("worker pool %s: instance_type is required", poolName))
		}

		// Only validate instance_count if autoscaling is not enabled
		if !pool.AutoscalingEnabled() {
			if pool.InstanceCount < 1 {
				poolName := "unknown"
				if pool.Name != nil {
					poolName = *pool.Name
				}
				v.errors = append(v.errors, fmt.Sprintf("worker pool %s: instance_count must be at least 1", poolName))
			}
		}

		// Validate autoscaling settings if enabled
		if pool.AutoscalingEnabled() {
			poolName := "unknown"
			if pool.Name != nil {
				poolName = *pool.Name
			}

			// min_instances can be 0 (pool starts with no nodes and scales up as needed)
			if pool.Autoscaling.MinInstances < 0 {
				v.errors = append(v.errors, fmt.Sprintf("worker pool %s: autoscaling min_instances cannot be negative", poolName))
			}

			if pool.Autoscaling.MaxInstances <= pool.Autoscaling.MinInstances {
				v.errors = append(v.errors, fmt.Sprintf("worker pool %s: autoscaling max_instances must be greater than min_instances", poolName))
			}
		}

		// Validate locations must be specified
		poolName := "unknown"
		if pool.Name != nil {
			poolName = *pool.Name
		}
		if len(pool.Locations) == 0 {
			v.errors = append(v.errors, fmt.Sprintf("worker pool %s: locations is required", poolName))
		}

		// Validate placement group if configured
		v.validatePlacementGroup(pool.PlacementGroup, fmt.Sprintf("worker pool %s", poolName))

		// Validate labels and taints
		v.validateNodePoolLabelsAndTaints(&pool.NodePool, "worker", pool.Name)
	}
}

// validateNodePoolLabelsAndTaints validates labels and taints for a node pool
func (v *Validator) validateNodePoolLabelsAndTaints(pool *NodePool, poolType string, poolName *string) {
	name := "unknown"
	if poolName != nil {
		name = *poolName
	}

	// Validate Kubernetes labels
	kubernetesLabels := pool.KubernetesLabels()
	for i, label := range kubernetesLabels {
		if label.Key == "" {
			v.errors = append(v.errors, fmt.Sprintf("%s pool %s: kubernetes.labels[%d] has empty key", poolType, name, i))
		}
		// Check for invalid characters in label key/value that could cause issues
		if strings.ContainsAny(label.Key, " \t\n\r\"'`$\\;&|<>()") {
			v.errors = append(v.errors, fmt.Sprintf("%s pool %s: kubernetes.labels key '%s' contains invalid characters", poolType, name, label.Key))
		}
		if strings.ContainsAny(label.Value, " \t\n\r\"'`$\\;&|<>()") {
			v.errors = append(v.errors, fmt.Sprintf("%s pool %s: kubernetes.labels value '%s' contains invalid characters", poolType, name, label.Value))
		}
	}

	// Validate Hetzner labels
	hetznerLabels := pool.HetznerLabels()
	for i, label := range hetznerLabels {
		if label.Key == "" {
			v.errors = append(v.errors, fmt.Sprintf("%s pool %s: hetzner.labels[%d] has empty key", poolType, name, i))
		}
		// Check for invalid characters in label key/value that could cause issues
		if strings.ContainsAny(label.Key, " \t\n\r\"'`$\\;&|<>()") {
			v.errors = append(v.errors, fmt.Sprintf("%s pool %s: hetzner.labels key '%s' contains invalid characters", poolType, name, label.Key))
		}
		if strings.ContainsAny(label.Value, " \t\n\r\"'`$\\;&|<>()") {
			v.errors = append(v.errors, fmt.Sprintf("%s pool %s: hetzner.labels value '%s' contains invalid characters", poolType, name, label.Value))
		}
	}

	// Validate Kubernetes taints
	validEffects := map[string]bool{
		"NoSchedule":       true,
		"PreferNoSchedule": true,
		"NoExecute":        true,
	}
	kubernetesTaints := pool.KubernetesTaints()
	for i, taint := range kubernetesTaints {
		if taint.Key == "" {
			v.errors = append(v.errors, fmt.Sprintf("%s pool %s: kubernetes.taints[%d] has empty key", poolType, name, i))
		}
		if taint.Effect == "" {
			v.errors = append(v.errors, fmt.Sprintf("%s pool %s: kubernetes.taints[%d] has empty effect", poolType, name, i))
		} else if !validEffects[taint.Effect] {
			v.errors = append(v.errors, fmt.Sprintf("%s pool %s: kubernetes.taints[%d] has invalid effect '%s' (must be NoSchedule, PreferNoSchedule, or NoExecute)", poolType, name, i, taint.Effect))
		}
		// Check for invalid characters in taint key/value that could cause issues
		if strings.ContainsAny(taint.Key, " \t\n\r\"'`$\\;&|<>()") {
			v.errors = append(v.errors, fmt.Sprintf("%s pool %s: kubernetes.taints key '%s' contains invalid characters", poolType, name, taint.Key))
		}
		if strings.ContainsAny(taint.Value, " \t\n\r\"'`$\\;&|<>()") {
			v.errors = append(v.errors, fmt.Sprintf("%s pool %s: kubernetes.taints value '%s' contains invalid characters", poolType, name, taint.Value))
		}
	}
}

// validateDatastore validates datastore configuration including S3 settings
func (v *Validator) validateDatastore() {
	// Validate datastore mode
	validModes := []string{"etcd", "external"}
	isValidMode := false
	for _, validMode := range validModes {
		if v.config.Datastore.Mode == validMode {
			isValidMode = true
			break
		}
	}
	if !isValidMode {
		v.errors = append(v.errors, fmt.Sprintf("datastore: invalid mode '%s', must be one of: %s",
			v.config.Datastore.Mode, strings.Join(validModes, ", ")))
	}

	// Validate embedded etcd configuration if mode is etcd
	if v.config.Datastore.Mode == "etcd" && v.config.Datastore.EmbeddedEtcd != nil {
		etcd := v.config.Datastore.EmbeddedEtcd

		// Validate snapshot retention
		if etcd.SnapshotRetention < 0 {
			v.errors = append(v.errors, "datastore.embedded_etcd: snapshot_retention cannot be negative")
		}

		// Validate S3 configuration if S3 is enabled
		if etcd.S3Enabled {
			if etcd.S3Endpoint == "" {
				v.errors = append(v.errors, "datastore.embedded_etcd: s3_endpoint is required when S3 is enabled")
			}
			if etcd.S3Region == "" {
				v.errors = append(v.errors, "datastore.embedded_etcd: s3_region is required when S3 is enabled")
			}
			if etcd.S3AccessKey == "" {
				v.errors = append(v.errors, "datastore.embedded_etcd: s3_access_key is required when S3 is enabled")
			}
			if etcd.S3SecretKey == "" {
				v.errors = append(v.errors, "datastore.embedded_etcd: s3_secret_key is required when S3 is enabled")
			}

			// Bucket name is required when S3 is enabled
			if etcd.S3Bucket == "" {
				v.errors = append(v.errors, "datastore.embedded_etcd: s3_bucket is required when S3 is enabled")
			} else {
				v.warnings = append(v.warnings, fmt.Sprintf("datastore.embedded_etcd: using S3 bucket '%s', ensure it exists before cluster creation", etcd.S3Bucket))
			}
		}
	}

	// Validate external datastore configuration if mode is external
	if v.config.Datastore.Mode == "external" && v.config.Datastore.ExternalDatastore != nil {
		external := v.config.Datastore.ExternalDatastore
		if external.Endpoint == "" {
			v.errors = append(v.errors, "datastore.external_datastore: endpoint is required when using external datastore")
		}
	}
}

// validateLoadBalancer validates load balancer configuration
func (v *Validator) validateLoadBalancer() {
	if !v.config.LoadBalancer.Enabled {
		return
	}

	// Validate load balancer type
	validTypes := []string{"lb11", "lb21", "lb31"}
	isValidType := false
	for _, validType := range validTypes {
		if v.config.LoadBalancer.Type == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		v.errors = append(v.errors, fmt.Sprintf("load_balancer: invalid type '%s', must be one of: %s",
			v.config.LoadBalancer.Type, strings.Join(validTypes, ", ")))
	}

	// Validate algorithm type
	validAlgorithms := []string{"round_robin", "least_connections"}
	isValidAlgorithm := false
	for _, validAlg := range validAlgorithms {
		if v.config.LoadBalancer.Algorithm.Type == validAlg {
			isValidAlgorithm = true
			break
		}
	}
	if !isValidAlgorithm {
		v.errors = append(v.errors, fmt.Sprintf("load_balancer: invalid algorithm type '%s', must be one of: %s",
			v.config.LoadBalancer.Algorithm.Type, strings.Join(validAlgorithms, ", ")))
	}

	// Validate services
	if len(v.config.LoadBalancer.Services) == 0 {
		v.warnings = append(v.warnings, "load_balancer: no services configured, using defaults (HTTP:80, HTTPS:443)")
	}

	for i, svc := range v.config.LoadBalancer.Services {
		// Validate protocol - tcp http https are supported
		validProtocols := []string{"tcp", "http", "https"}
		isValidProtocol := false
		for _, validProto := range validProtocols {
			if svc.Protocol == validProto {
				isValidProtocol = true
				break
			}
		}
		if !isValidProtocol {
			v.errors = append(v.errors, fmt.Sprintf("load_balancer: service %d has invalid protocol '%s', must be one of: %s",
				i+1, svc.Protocol, strings.Join(validProtocols, ", ")))
		}

		// Validate ports
		if svc.ListenPort < 1 || svc.ListenPort > 65535 {
			v.errors = append(v.errors, fmt.Sprintf("load_balancer: service %d has invalid listen_port %d, must be between 1 and 65535",
				i+1, svc.ListenPort))
		}
		if svc.DestinationPort < 1 || svc.DestinationPort > 65535 {
			v.errors = append(v.errors, fmt.Sprintf("load_balancer: service %d has invalid destination_port %d, must be between 1 and 65535",
				i+1, svc.DestinationPort))
		}

		// Validate health check if present
		if svc.HealthCheck != nil {
			if svc.HealthCheck.Port < 1 || svc.HealthCheck.Port > 65535 {
				v.errors = append(v.errors, fmt.Sprintf("load_balancer: service %d health check has invalid port %d",
					i+1, svc.HealthCheck.Port))
			}
			if svc.HealthCheck.Interval < 3 || svc.HealthCheck.Interval > 3600 {
				v.errors = append(v.errors, fmt.Sprintf("load_balancer: service %d health check interval must be between 3 and 3600 seconds",
					i+1))
			}
			if svc.HealthCheck.Timeout < 1 || svc.HealthCheck.Timeout > 3600 {
				v.errors = append(v.errors, fmt.Sprintf("load_balancer: service %d health check timeout must be between 1 and 3600 seconds",
					i+1))
			}
			if svc.HealthCheck.Retries < 1 || svc.HealthCheck.Retries > 10 {
				v.errors = append(v.errors, fmt.Sprintf("load_balancer: service %d health check retries must be between 1 and 10",
					i+1))
			}
		}
	}

	// Validate target pools if specified
	if len(v.config.LoadBalancer.TargetPools) > 0 {
		// Check that specified pools exist in worker pools
		workerPoolNames := make(map[string]bool)
		for _, pool := range v.config.WorkerNodePools {
			if pool.Name != nil {
				workerPoolNames[*pool.Name] = true
			}
		}

		for _, targetPool := range v.config.LoadBalancer.TargetPools {
			if !workerPoolNames[targetPool] {
				v.warnings = append(v.warnings, fmt.Sprintf("load_balancer: target_pool '%s' not found in worker_node_pools", targetPool))
			}
		}
	} else {
		// If no target pools specified, all worker nodes will be used
		if len(v.config.WorkerNodePools) == 0 {
			v.warnings = append(v.warnings, "load_balancer: no target_pools specified and no worker_node_pools configured")
		}
	}

	// Validate use_private_ip setting
	if v.config.LoadBalancer.UsePrivateIP != nil && *v.config.LoadBalancer.UsePrivateIP && !v.config.Networking.PrivateNetwork.Enabled {
		v.errors = append(v.errors, "load_balancer: use_private_ip requires private_network to be enabled")
	}

	// Validate attach_to_network setting
	if v.config.LoadBalancer.AttachToNetwork && !v.config.Networking.PrivateNetwork.Enabled {
		v.errors = append(v.errors, "load_balancer: attach_to_network requires private_network to be enabled")
	}
}

// validateExternalTools validates that required external tools are available
func (v *Validator) validateExternalTools() {
	// Note: External tools (kubectl, helm) are now automatically installed
	// by the tool installer before cluster operations, so we don't need to
	// treat missing tools as errors here. This method is kept for potential
	// future validation needs.
}

// validateDomain validates the domain format
func (v *Validator) validateDomain() {
	if v.config.Domain == "" {
		// Domain is optional, no error if not set
		return
	}

	// Domain should be a valid DNS name
	// Allow domains like example.com, subdomain.example.com, etc.
	validDomain := regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$`)
	if !validDomain.MatchString(v.config.Domain) {
		v.errors = append(v.errors,
			"domain must be a valid DNS name (e.g., example.com, subdomain.example.com)")
	}

	if len(v.config.Domain) > 253 {
		v.errors = append(v.errors, "domain must be 253 characters or less")
	}
}

// validateDNSZone validates DNS zone configuration
func (v *Validator) validateDNSZone() {
	if !v.config.DNSZone.Enabled {
		// DNS zone is disabled, no validation needed
		return
	}

	// If DNS zone is enabled, domain must be set
	if v.config.Domain == "" {
		v.errors = append(v.errors,
			"domain is required when dns_zone.enabled is true")
	}

	// If DNS zone is enabled, global load balancer should be enabled
	if !v.config.LoadBalancer.Enabled {
		v.warnings = append(v.warnings,
			"dns_zone is enabled but load_balancer is not enabled. DNS records will not be created.")
	}

	// Validate TTL
	if v.config.DNSZone.TTL < 60 {
		v.errors = append(v.errors,
			"dns_zone.ttl must be at least 60 seconds")
	}

	if v.config.DNSZone.TTL > 86400 {
		v.warnings = append(v.warnings,
			"dns_zone.ttl is very high (>24 hours), consider using a lower value for faster DNS propagation")
	}
}

// validateSSLCertificate validates SSL certificate configuration
func (v *Validator) validateSSLCertificate() {
	if !v.config.SSLCertificate.Enabled {
		// SSL certificate is disabled, no validation needed
		return
	}

	// If SSL certificate is enabled, DNS zone must be enabled for managed certificates
	if !v.config.DNSZone.Enabled {
		v.errors = append(v.errors,
			"dns_zone.enabled must be true when ssl_certificate.enabled is true (required for DNS validation)")
	}

	// If SSL certificate is enabled, domain must be set
	if v.config.Domain == "" {
		v.errors = append(v.errors,
			"domain is required when ssl_certificate.enabled is true")
	}

	// If SSL certificate is enabled, global load balancer must be enabled
	if !v.config.LoadBalancer.Enabled {
		v.errors = append(v.errors,
			"load_balancer.enabled must be true when ssl_certificate.enabled is true")
	}

	// Check if load balancer has HTTPS service
	hasHTTPSService := false
	for _, svc := range v.config.LoadBalancer.Services {
		if strings.ToLower(svc.Protocol) == "https" {
			hasHTTPSService = true
			break
		}
	}

	if !hasHTTPSService {
		v.warnings = append(v.warnings,
			"ssl_certificate is enabled but no HTTPS service found in load_balancer.services. Certificate will be created but not used.")
	}
}

// validatePlacementGroup validates placement group configuration
func (v *Validator) validatePlacementGroup(pg *PlacementGroupConfig, context string) {
	if pg == nil {
		return
	}

	if pg.Name == "" {
		v.errors = append(v.errors, fmt.Sprintf("%s: placement_group.name is required", context))
	}

	if pg.Type == "" {
		v.errors = append(v.errors, fmt.Sprintf("%s: placement_group.type is required", context))
	} else if pg.Type != "spread" {
		v.errors = append(v.errors, fmt.Sprintf("%s: placement_group.type must be 'spread' (got '%s')", context, pg.Type))
	}

	// Validate labels
	for i, label := range pg.Labels {
		if label.Key == "" {
			v.errors = append(v.errors, fmt.Sprintf("%s: placement_group.labels[%d] has empty key", context, i))
		}
	}
}

// GetErrors returns validation errors
func (v *Validator) GetErrors() []string {
	return v.errors
}

// GetWarnings returns validation warnings
func (v *Validator) GetWarnings() []string {
	return v.warnings
}
