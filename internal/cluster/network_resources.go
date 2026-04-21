// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package cluster

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/internal/util"
	"github.com/magenx/kuberaptor/pkg/hetzner"
)

// NetworkResourceManager handles creation of firewalls and load balancers
type NetworkResourceManager struct {
	Config        *config.Main
	HetznerClient *hetzner.Client
	ctx           context.Context
}

// NewNetworkResourceManager creates a new network resource manager
func NewNetworkResourceManager(cfg *config.Main, hetznerClient *hetzner.Client) *NetworkResourceManager {
	return &NetworkResourceManager{
		Config:        cfg,
		HetznerClient: hetznerClient,
		ctx:           context.Background(),
	}
}

// CreateAPILoadBalancers creates multiple API load balancers for multi-location deployment
// Each location gets its own API load balancer targeting only masters in that location
func (n *NetworkResourceManager) CreateAPILoadBalancers(masterServers []*hcloud.Server, locations []string, network *hcloud.Network) ([]*hcloud.LoadBalancer, error) {
	loadBalancers := make([]*hcloud.LoadBalancer, 0, len(locations))

	for _, location := range locations {
		// Create API load balancer for this location
		lb, err := n.createAPILoadBalancerForLocation(masterServers, location, network)
		if err != nil {
			return nil, fmt.Errorf("failed to create API load balancer for location %s: %w", location, err)
		}
		loadBalancers = append(loadBalancers, lb)
	}

	util.LogSuccess(fmt.Sprintf("Created %d API load balancer(s) across locations", len(loadBalancers)), "load balancer")
	return loadBalancers, nil
}

// createAPILoadBalancerForLocation creates an API load balancer for a specific location
func (n *NetworkResourceManager) createAPILoadBalancerForLocation(masterServers []*hcloud.Server, location string, network *hcloud.Network) (*hcloud.LoadBalancer, error) {
	// Name includes location suffix for multi-location deployments
	lbName := fmt.Sprintf("%s-api-lb-%s", n.Config.ClusterName, location)

	// Check if load balancer already exists
	existingLB, err := n.HetznerClient.GetLoadBalancer(n.ctx, lbName)
	if err == nil && existingLB != nil {
		util.LogInfo(fmt.Sprintf("API load balancer already exists for %s, using existing load balancer", location), "load balancer")
		return existingLB, nil
	}

	util.LogInfo(fmt.Sprintf("Creating API load balancer in %s", location), "load balancer")

	// Determine if API load balancer should be attached to network
	shouldAttachToNetwork := n.Config.Networking.PrivateNetwork.Enabled && network != nil

	// Create load balancer without targets first to avoid API validation issues
	opts := hcloud.LoadBalancerCreateOpts{
		Name:             lbName,
		LoadBalancerType: &hcloud.LoadBalancerType{Name: "lb11"}, // Smallest LB type
		Location:         &hcloud.Location{Name: location},
		Labels:           n.buildAPILoadBalancerLabels(location),
		Services: []hcloud.LoadBalancerCreateOptsService{
			{
				Protocol:        hcloud.LoadBalancerServiceProtocolTCP,
				ListenPort:      hcloud.Ptr(6443),
				DestinationPort: hcloud.Ptr(6443),
				HealthCheck: &hcloud.LoadBalancerCreateOptsServiceHealthCheck{
					Protocol: hcloud.LoadBalancerServiceProtocolTCP,
					Port:     hcloud.Ptr(6443),
					Interval: hcloud.Duration(15 * time.Second),
					Timeout:  hcloud.Duration(10 * time.Second),
					Retries:  hcloud.Ptr(3),
				},
			},
		},
		PublicInterface: hcloud.Ptr(true),
	}

	// Attach to network if private network is enabled
	if shouldAttachToNetwork {
		opts.Network = network
	}

	lb, err := n.HetznerClient.CreateLoadBalancer(n.ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create load balancer: %w", err)
	}

	util.LogSuccess(fmt.Sprintf("API load balancer created in %s: %s (IP: %s)", location, lbName, lb.PublicNet.IPv4.IP.String()), "load balancer")

	// Apply deletion protection if configured
	if n.Config.ProtectAgainstDeletion {
		if err := n.HetznerClient.ChangeLoadBalancerProtection(n.ctx, lb, true); err != nil {
			return nil, fmt.Errorf("failed to enable protection for API load balancer %s: %w", lbName, err)
		}
	}

	// Default retry configuration for network attachment verification
	const (
		maxNetworkAttachmentRetries = 5
		initialRetryDelay           = 2 * time.Second
		stabilizationDelay          = 5 * time.Second
	)

	// If load balancer was created with a network attachment, refresh its data
	if shouldAttachToNetwork {
		util.LogInfo("Refreshing load balancer data to verify network attachment", "load balancer")

		retryDelay := initialRetryDelay
		for attempt := 0; attempt < maxNetworkAttachmentRetries; attempt++ {
			if attempt > 0 {
				time.Sleep(retryDelay)
				retryDelay *= 2
			}

			refreshedLB, err := n.HetznerClient.GetLoadBalancer(n.ctx, lbName)
			if err != nil {
				return nil, fmt.Errorf("failed to refresh load balancer: %w", err)
			}
			if refreshedLB == nil {
				return nil, fmt.Errorf("load balancer %s not found after creation", lbName)
			}

			if len(refreshedLB.PrivateNet) > 0 {
				lb = refreshedLB
				util.LogInfo("Load balancer network attachment verified", "load balancer")
				util.LogInfo("Waiting for network attachment to stabilize", "load balancer")
				time.Sleep(stabilizationDelay)
				break
			}

			if attempt == maxNetworkAttachmentRetries-1 {
				return nil, fmt.Errorf("load balancer network attachment not complete after %d attempts", maxNetworkAttachmentRetries)
			}

			util.LogInfo(fmt.Sprintf("Network attachment in progress, retrying... (attempt %d/%d)", attempt+2, maxNetworkAttachmentRetries), "load balancer")
		}
	} else {
		// Add a stabilization delay to ensure the load balancer is fully operational
		util.LogInfo("Waiting for load balancer to stabilize before adding targets", "load balancer")
		time.Sleep(stabilizationDelay)
	}

	// Add master servers as targets using label selector for this location only
	util.LogInfo(fmt.Sprintf("Adding master servers in %s as targets to API load balancer", location), "load balancer")
	labelSelector := fmt.Sprintf("role=master,cluster=%s,location=%s", n.Config.ClusterName, location)

	// Use private IP if network is attached, otherwise use public IP
	usePrivateIP := shouldAttachToNetwork

	err = n.addLabelSelectorTargetWithRetry(lbName, lb, location, hcloud.LoadBalancerAddLabelSelectorTargetOpts{
		Selector:     labelSelector,
		UsePrivateIP: hcloud.Ptr(usePrivateIP),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add master servers as targets in %s: %w", location, err)
	}

	util.LogSuccess(fmt.Sprintf("Master servers in %s added as targets to API load balancer", location), "load balancer")

	return lb, nil
}

// CreateClusterFirewall creates a firewall for the cluster
func (n *NetworkResourceManager) CreateClusterFirewall(network *hcloud.Network) (*hcloud.Firewall, error) {
	util.LogInfo("Creating firewall for cluster", "firewall")

	fwName := fmt.Sprintf("%s-firewall", n.Config.ClusterName)

	// Define firewall rules
	var rules []hcloud.FirewallRule

	// Allow SSH from configured networks
	if len(n.Config.Networking.AllowedNetworks.SSH) > 0 {
		for _, cidr := range n.Config.Networking.AllowedNetworks.SSH {
			rules = append(rules, hcloud.FirewallRule{
				Direction:   hcloud.FirewallRuleDirectionIn,
				SourceIPs:   []net.IPNet{parseCIDR(cidr)},
				Protocol:    hcloud.FirewallRuleProtocolTCP,
				Port:        hcloud.Ptr(fmt.Sprintf("%d", n.Config.Networking.SSH.Port)),
				Description: hcloud.Ptr("SSH access"),
			})
		}
	}

	// Allow Kubernetes API from configured networks
	if len(n.Config.Networking.AllowedNetworks.API) > 0 {
		for _, cidr := range n.Config.Networking.AllowedNetworks.API {
			rules = append(rules, hcloud.FirewallRule{
				Direction:   hcloud.FirewallRuleDirectionIn,
				SourceIPs:   []net.IPNet{parseCIDR(cidr)},
				Protocol:    hcloud.FirewallRuleProtocolTCP,
				Port:        hcloud.Ptr("6443"),
				Description: hcloud.Ptr("Kubernetes API access"),
			})
		}
	}

	// Allow all traffic within private network if enabled
	if n.Config.Networking.PrivateNetwork.Enabled {
		_, privateNet, err := net.ParseCIDR(n.Config.Networking.PrivateNetwork.Subnet)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private network subnet: %w", err)
		}
		rules = append(rules, hcloud.FirewallRule{
			Direction:   hcloud.FirewallRuleDirectionIn,
			SourceIPs:   []net.IPNet{*privateNet},
			Protocol:    hcloud.FirewallRuleProtocolTCP,
			Port:        hcloud.Ptr("1-65535"),
			Description: hcloud.Ptr("Allow all TCP within cluster network"),
		})
		rules = append(rules, hcloud.FirewallRule{
			Direction:   hcloud.FirewallRuleDirectionIn,
			SourceIPs:   []net.IPNet{*privateNet},
			Protocol:    hcloud.FirewallRuleProtocolUDP,
			Port:        hcloud.Ptr("1-65535"),
			Description: hcloud.Ptr("Allow all UDP within cluster network"),
		})
		rules = append(rules, hcloud.FirewallRule{
			Direction:   hcloud.FirewallRuleDirectionIn,
			SourceIPs:   []net.IPNet{*privateNet},
			Protocol:    hcloud.FirewallRuleProtocolICMP,
			Description: hcloud.Ptr("Allow ICMP within cluster network"),
		})
	}

	// Create firewall
	fw, err := n.HetznerClient.CreateFirewall(n.ctx, hcloud.FirewallCreateOpts{
		Name: fwName,
		Labels: map[string]string{
			"cluster": n.Config.ClusterName,
		},
		Rules: rules,
		ApplyTo: []hcloud.FirewallResource{
			{
				Type: hcloud.FirewallResourceTypeLabelSelector,
				LabelSelector: &hcloud.FirewallResourceLabelSelector{
					Selector: fmt.Sprintf("cluster=%s", n.Config.ClusterName),
				},
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create firewall: %w", err)
	}

	util.LogSuccess(fmt.Sprintf("Firewall created: %s with %d rule(s)", fwName, len(rules)), "firewall")

	return fw, nil
}

// CreateGlobalLoadBalancers creates multiple global load balancers for multi-location deployment
// This enables true location autonomy with one load balancer per region
func (n *NetworkResourceManager) CreateGlobalLoadBalancers(network *hcloud.Network, locations []string, certificate *hcloud.Certificate) ([]*hcloud.LoadBalancer, error) {
	// Skip if load balancer is not enabled
	if !n.Config.LoadBalancer.Enabled {
		util.LogInfo("Global load balancer is disabled, skipping creation", "load balancer")
		return nil, nil
	}

	loadBalancers := make([]*hcloud.LoadBalancer, 0, len(locations))

	for _, location := range locations {
		// Create load balancer for this location
		lb, err := n.createGlobalLoadBalancerForLocation(network, location, certificate)
		if err != nil {
			return nil, fmt.Errorf("failed to create load balancer for location %s: %w", location, err)
		}
		loadBalancers = append(loadBalancers, lb)
	}

	util.LogSuccess(fmt.Sprintf("Created %d global load balancer(s) across locations", len(loadBalancers)), "load balancer")
	return loadBalancers, nil
}

// createGlobalLoadBalancerForLocation creates a load balancer for a specific location
func (n *NetworkResourceManager) createGlobalLoadBalancerForLocation(network *hcloud.Network, location string, certificate *hcloud.Certificate) (*hcloud.LoadBalancer, error) {
	// Determine load balancer name with location suffix
	// Pattern: {cluster-name}-global-lb-{location} or {cluster-name}-{custom-name}-global-lb-{location}
	var lbName string
	if n.Config.LoadBalancer.Name != nil && *n.Config.LoadBalancer.Name != "" {
		// Include custom name in the pattern
		lbName = fmt.Sprintf("%s-%s-global-lb-%s", n.Config.ClusterName, *n.Config.LoadBalancer.Name, location)
	} else {
		// Default pattern without custom name
		lbName = fmt.Sprintf("%s-global-lb-%s", n.Config.ClusterName, location)
	}

	// Check if load balancer already exists
	existingLB, err := n.HetznerClient.GetLoadBalancer(n.ctx, lbName)
	if err == nil && existingLB != nil {
		util.LogInfo(fmt.Sprintf("Global load balancer already exists for %s, using existing load balancer", location), "load balancer")
		return existingLB, nil
	}

	util.LogInfo(fmt.Sprintf("Creating global load balancer in %s", location), "load balancer")

	// Build services configuration
	var services []hcloud.LoadBalancerCreateOptsService
	for _, svc := range n.Config.LoadBalancer.Services {
		service := hcloud.LoadBalancerCreateOptsService{
			Protocol:        hcloud.LoadBalancerServiceProtocol(svc.Protocol),
			ListenPort:      hcloud.Ptr(svc.ListenPort),
			DestinationPort: hcloud.Ptr(svc.DestinationPort),
			Proxyprotocol:   hcloud.Ptr(svc.ProxyProtocol),
		}

		// Add HTTP configuration for HTTPS services with certificate
		if strings.ToLower(svc.Protocol) == "https" && certificate != nil {
			httpConfig := &hcloud.LoadBalancerCreateOptsServiceHTTP{
				Certificates: []*hcloud.Certificate{certificate},
			}
			service.HTTP = httpConfig
			util.LogInfo(fmt.Sprintf("Attaching SSL certificate to HTTPS service on port %d", svc.ListenPort), "load balancer")
		}

		// Add health check if configured
		if svc.HealthCheck != nil {
			healthCheck := &hcloud.LoadBalancerCreateOptsServiceHealthCheck{
				Protocol: hcloud.LoadBalancerServiceProtocol(svc.HealthCheck.Protocol),
				Port:     hcloud.Ptr(svc.HealthCheck.Port),
				Interval: hcloud.Duration(time.Duration(svc.HealthCheck.Interval) * time.Second),
				Timeout:  hcloud.Duration(time.Duration(svc.HealthCheck.Timeout) * time.Second),
				Retries:  hcloud.Ptr(svc.HealthCheck.Retries),
			}

			// Add HTTP-specific health check settings
			if svc.HealthCheck.HTTP != nil {
				healthCheck.HTTP = &hcloud.LoadBalancerCreateOptsServiceHealthCheckHTTP{
					Domain:      hcloud.Ptr(svc.HealthCheck.HTTP.Domain),
					Path:        hcloud.Ptr(svc.HealthCheck.HTTP.Path),
					StatusCodes: svc.HealthCheck.HTTP.StatusCodes,
					TLS:         hcloud.Ptr(svc.HealthCheck.HTTP.TLS),
				}
			}

			service.HealthCheck = healthCheck
		}

		services = append(services, service)
	}

	// Determine if load balancer should be attached to network
	shouldAttachToNetwork := n.Config.LoadBalancer.AttachToNetwork && n.Config.Networking.PrivateNetwork.Enabled && network != nil

	// Determine if we should use private IP for targets
	usePrivateIP := false
	if n.Config.LoadBalancer.UsePrivateIP != nil {
		usePrivateIP = *n.Config.LoadBalancer.UsePrivateIP && shouldAttachToNetwork
	} else if shouldAttachToNetwork {
		usePrivateIP = true
	}

	// Build create options without targets
	opts := hcloud.LoadBalancerCreateOpts{
		Name:             lbName,
		LoadBalancerType: &hcloud.LoadBalancerType{Name: n.Config.LoadBalancer.Type},
		Location:         &hcloud.Location{Name: location},
		Labels: map[string]string{
			"cluster":  n.Config.ClusterName,
			"role":     "global-lb",
			"location": location,
			"managed":  "kuberaptor",
		},
		Algorithm: &hcloud.LoadBalancerAlgorithm{
			Type: hcloud.LoadBalancerAlgorithmType(n.Config.LoadBalancer.Algorithm.Type),
		},
		Services:        services,
		PublicInterface: hcloud.Ptr(true),
	}

	// Attach to network if configured
	if shouldAttachToNetwork {
		opts.Network = network
	}

	// Create load balancer
	lb, err := n.HetznerClient.CreateLoadBalancer(n.ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create load balancer: %w", err)
	}

	util.LogSuccess(fmt.Sprintf("Global load balancer created in %s: %s (IP: %s)", location, lbName, lb.PublicNet.IPv4.IP.String()), "load balancer")

	// Apply deletion protection if configured
	if n.Config.ProtectAgainstDeletion {
		if err := n.HetznerClient.ChangeLoadBalancerProtection(n.ctx, lb, true); err != nil {
			return nil, fmt.Errorf("failed to enable protection for load balancer %s: %w", lbName, err)
		}
	}

	// Default retry configuration for network attachment verification
	const (
		maxNetworkAttachmentRetries = 5
		initialRetryDelay           = 2 * time.Second
		stabilizationDelay          = 5 * time.Second
	)

	// If load balancer was created with a network attachment, refresh its data
	if shouldAttachToNetwork {
		util.LogInfo("Refreshing load balancer data to verify network attachment", "load balancer")

		retryDelay := initialRetryDelay
		for attempt := 0; attempt < maxNetworkAttachmentRetries; attempt++ {
			if attempt > 0 {
				time.Sleep(retryDelay)
				retryDelay *= 2
			}

			refreshedLB, err := n.HetznerClient.GetLoadBalancer(n.ctx, lbName)
			if err != nil {
				return nil, fmt.Errorf("failed to refresh load balancer: %w", err)
			}
			if refreshedLB == nil {
				return nil, fmt.Errorf("load balancer %s not found after creation", lbName)
			}

			if len(refreshedLB.PrivateNet) > 0 {
				lb = refreshedLB
				util.LogInfo("Load balancer network attachment verified", "load balancer")
				util.LogInfo("Waiting for network attachment to stabilize", "load balancer")
				time.Sleep(stabilizationDelay)
				break
			}

			if attempt == maxNetworkAttachmentRetries-1 {
				return nil, fmt.Errorf("load balancer network attachment not complete after %d attempts", maxNetworkAttachmentRetries)
			}

			util.LogInfo(fmt.Sprintf("Network attachment in progress, retrying... (attempt %d/%d)", attempt+2, maxNetworkAttachmentRetries), "load balancer")
		}
	}

	// Add label selector targets - only target workers in this location for regional autonomy
	util.LogInfo(fmt.Sprintf("Adding targets in %s to load balancer", location), "load balancer")

	if len(n.Config.LoadBalancer.TargetPools) > 0 {
		// Add a separate label selector target for each pool in this location
		for _, poolName := range n.Config.LoadBalancer.TargetPools {
			labelSelector := fmt.Sprintf("pool=%s,location=%s", poolName, location)
			err = n.addLabelSelectorTargetWithRetry(lbName, lb, location, hcloud.LoadBalancerAddLabelSelectorTargetOpts{
				Selector:     labelSelector,
				UsePrivateIP: hcloud.Ptr(usePrivateIP),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add targets to load balancer for pool %s in %s: %w", poolName, location, err)
			}
		}
	} else {
		// Default to all worker nodes in the cluster for this location only
		labelSelector := fmt.Sprintf("role=worker,cluster=%s,location=%s", n.Config.ClusterName, location)
		err = n.addLabelSelectorTargetWithRetry(lbName, lb, location, hcloud.LoadBalancerAddLabelSelectorTargetOpts{
			Selector:     labelSelector,
			UsePrivateIP: hcloud.Ptr(usePrivateIP),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add targets to load balancer in %s: %w", location, err)
		}
	}

	util.LogSuccess(fmt.Sprintf("Targets in %s added to load balancer", location), "load balancer")

	return lb, nil
}

// CreateDNSZone creates a DNS zone in Hetzner for the domain
func (n *NetworkResourceManager) CreateDNSZone() (*hcloud.Zone, error) {
	// Skip if DNS zone is not enabled
	if !n.Config.DNSZone.Enabled {
		util.LogInfo("DNS zone creation is disabled, skipping", "dns")
		return nil, nil
	}

	// Skip if domain is not set
	if n.Config.Domain == "" {
		util.LogInfo("Domain is not set, skipping DNS zone creation", "dns")
		return nil, nil
	}

	util.LogInfo(fmt.Sprintf("Creating DNS zone for domain: %s", n.Config.Domain), "dns")

	// Determine zone name (use override if provided, otherwise use domain)
	zoneName := n.Config.Domain
	if n.Config.DNSZone.Name != "" {
		zoneName = n.Config.DNSZone.Name
	}

	// Check if zone already exists
	existingZone, err := n.HetznerClient.GetZone(n.ctx, zoneName)
	if err == nil && existingZone != nil {
		util.LogInfo(fmt.Sprintf("DNS zone already exists: %s", zoneName), "dns")
		return existingZone, nil
	}

	// Create DNS zone
	zone, err := n.HetznerClient.CreateZone(n.ctx, hcloud.ZoneCreateOpts{
		Name: zoneName,
		Mode: hcloud.ZoneModePrimary,
		TTL:  hcloud.Ptr(n.Config.DNSZone.TTL),
		Labels: map[string]string{
			"cluster": n.Config.ClusterName,
			"managed": "kuberaptor",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create DNS zone: %w", err)
	}

	util.LogSuccess(fmt.Sprintf("DNS zone created: %s", zoneName), "dns")

	// Display nameservers information
	if len(zone.AuthoritativeNameservers.Assigned) > 0 {
		util.LogInfo("DNS zone nameservers:", "dns")
		for _, ns := range zone.AuthoritativeNameservers.Assigned {
			util.LogInfo(fmt.Sprintf("  - %s", ns), "dns")
		}
		util.LogInfo(fmt.Sprintf("Update your domain registrar to use these nameservers for domain: %s", n.Config.Domain), "dns")
	}

	return zone, nil
}

// parseCIDR parses a CIDR string and returns net.IPNet
func parseCIDR(cidr string) net.IPNet {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		// Return a restrictive default that won't match anything if parsing fails
		// This is safer than allowing 0.0.0.0/0
		_, ipnet, _ = net.ParseCIDR("127.0.0.1/32")
		util.LogWarning(fmt.Sprintf("Failed to parse CIDR %s, using restrictive default", cidr), "firewall")
	}
	return *ipnet
}

// CreateSSLCertificate creates a managed SSL certificate for the domain
// The certificate will cover both the root domain and wildcard subdomain
func (n *NetworkResourceManager) CreateSSLCertificate() (*hcloud.Certificate, error) {
	// Skip if SSL certificate is not enabled
	if !n.Config.SSLCertificate.Enabled {
		util.LogInfo("SSL certificate creation is disabled, skipping", "ssl")
		return nil, nil
	}

	// Skip if domain is not set
	if n.Config.Domain == "" {
		util.LogInfo("Domain is not set, skipping SSL certificate creation", "ssl")
		return nil, nil
	}

	util.LogInfo(fmt.Sprintf("Creating managed SSL certificate for domain: %s", n.Config.Domain), "ssl")

	// Determine certificate name (use override if provided, otherwise use domain)
	certName := n.Config.Domain
	if n.Config.SSLCertificate.Name != "" {
		certName = n.Config.SSLCertificate.Name
	}

	// Determine domain for certificate (use override if provided, otherwise use domain from config)
	certDomain := n.Config.Domain
	if n.Config.SSLCertificate.Domain != "" {
		certDomain = n.Config.SSLCertificate.Domain
	}

	// Check if certificate already exists
	existingCert, err := n.HetznerClient.GetCertificate(n.ctx, certName)
	if err == nil && existingCert != nil {
		util.LogInfo(fmt.Sprintf("SSL certificate already exists: %s", certName), "ssl")
		return existingCert, nil
	}

	// Create managed certificate with root domain and wildcard
	// This allows the certificate to be used for both example.com and *.example.com
	domainNames := []string{
		certDomain,
		fmt.Sprintf("*.%s", certDomain),
	}

	cert, err := n.HetznerClient.CreateManagedCertificate(n.ctx, hcloud.CertificateCreateOpts{
		Name:        certName,
		Type:        hcloud.CertificateTypeManaged,
		DomainNames: domainNames,
		Labels: map[string]string{
			"cluster": n.Config.ClusterName,
			"managed": "kuberaptor",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create SSL certificate: %w", err)
	}

	util.LogSuccess(fmt.Sprintf("SSL certificate created: %s (covers: %s)", certName, strings.Join(domainNames, ", ")), "ssl")
	util.LogInfo("Certificate validation will happen in the background (may take up to 5 minutes)", "ssl")
	util.LogInfo("The certificate will be automatically validated via DNS records in your DNS zone", "ssl")

	return cert, nil
}

func (n *NetworkResourceManager) addLabelSelectorTargetWithRetry(lbName string, lb *hcloud.LoadBalancer, location string, opts hcloud.LoadBalancerAddLabelSelectorTargetOpts) error {
	const (
		maxRetries         = 6
		initialRetryDelay  = 2 * time.Second
		maxRetryDelay      = 20 * time.Second
		stabilizationDelay = 3 * time.Second
	)

	retryDelay := initialRetryDelay
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = n.HetznerClient.AddLabelSelectorTargetToLoadBalancer(n.ctx, lb, opts)
		if err == nil {
			return nil
		}

		if !isRetryableLoadBalancerTargetError(err) || attempt == maxRetries {
			return err
		}

		util.LogInfo(
			fmt.Sprintf("Target attachment not yet ready for %s in %s, retrying...", lbName, location),
			"load balancer",
		)

		time.Sleep(retryDelay)

		// Refresh load balancer state between retries to pick up eventual-consistency updates,
		// especially network attachment metadata that can lag right after creation.
		refreshedLB, refreshErr := n.HetznerClient.GetLoadBalancer(n.ctx, lbName)
		if refreshErr == nil && refreshedLB != nil {
			lb = refreshedLB
			if len(refreshedLB.PrivateNet) > 0 {
				// When a private network is attached, wait briefly so target attachment calls
				// observe the attachment state consistently across the API backend.
				time.Sleep(stabilizationDelay)
			}
		}

		retryDelay *= 2
		if retryDelay > maxRetryDelay {
			retryDelay = maxRetryDelay
		}
	}

	return err
}

func isRetryableLoadBalancerTargetError(err error) bool {
	return hcloud.IsError(
		err,
		hcloud.ErrorCodeLoadBalancerNotAttachedToNetwork,
		hcloud.ErrorCodeServerNotAttachedToNetwork,
		hcloud.ErrorCodeResourceUnavailable,
		hcloud.ErrorCodeConflict,
		hcloud.ErrorCodeTimeout,
	)
}

// buildAPILoadBalancerLabels builds the Hetzner Cloud labels for an API load balancer
// This merges the default labels (cluster, role, location, managed) with custom labels from the configuration
func (n *NetworkResourceManager) buildAPILoadBalancerLabels(location string) map[string]string {
	// Start with default labels
	labels := map[string]string{
		"cluster":  n.Config.ClusterName,
		"role":     "api-lb",
		"location": location,
		"managed":  "kuberaptor",
	}

	// Add custom Hetzner labels from API load balancer configuration
	if n.Config.APILoadBalancer.Hetzner != nil {
		customLabels := n.Config.APILoadBalancer.Hetzner.Labels
		for _, label := range customLabels {
			// Custom labels can override defaults except for "managed" which is always set
			if label.Key != "managed" {
				labels[label.Key] = label.Value
			}
		}
	}

	return labels
}
