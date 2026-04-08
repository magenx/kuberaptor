// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package hetzner

import (
	"context"
	"fmt"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/pkg/version"
)

// Client wraps the Hetzner Cloud client
type Client struct {
	hcloud *hcloud.Client
	token  string
}

// NewClient creates a new Hetzner client
func NewClient(token string) *Client {
	opts := []hcloud.ClientOption{
		hcloud.WithToken(token),
		hcloud.WithApplication("kuberaptor", version.Get()),
	}

	return &Client{
		hcloud: hcloud.NewClient(opts...),
		token:  token,
	}
}

// GetLocations returns all available locations
func (c *Client) GetLocations(ctx context.Context) ([]*hcloud.Location, error) {
	locations, err := c.hcloud.Location.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch locations: %w", err)
	}
	return locations, nil
}

// GetServerTypes returns all available server types
func (c *Client) GetServerTypes(ctx context.Context) ([]*hcloud.ServerType, error) {
	serverTypes, err := c.hcloud.ServerType.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch server types: %w", err)
	}
	return serverTypes, nil
}

// GetServerType returns a specific server type by name
func (c *Client) GetServerType(ctx context.Context, name string) (*hcloud.ServerType, error) {
	serverType, _, err := c.hcloud.ServerType.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch server type %s: %w", name, err)
	}
	if serverType == nil {
		return nil, fmt.Errorf("server type %s not found", name)
	}
	return serverType, nil
}

// GetLocation returns a specific location by name
func (c *Client) GetLocation(ctx context.Context, name string) (*hcloud.Location, error) {
	location, _, err := c.hcloud.Location.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch location %s: %w", name, err)
	}
	if location == nil {
		return nil, fmt.Errorf("location %s not found", name)
	}
	return location, nil
}

// GetImage returns a specific image by name or ID
func (c *Client) GetImage(ctx context.Context, nameOrID string) (*hcloud.Image, error) {
	image, _, err := c.hcloud.Image.GetByName(ctx, nameOrID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image %s: %w", nameOrID, err)
	}
	if image == nil {
		return nil, fmt.Errorf("image %s not found", nameOrID)
	}
	return image, nil
}

// ListServers returns all servers matching the label selector
func (c *Client) ListServers(ctx context.Context, opts hcloud.ServerListOpts) ([]*hcloud.Server, error) {
	servers, err := c.hcloud.Server.AllWithOpts(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}
	return servers, nil
}

// GetServer returns a specific server by name
func (c *Client) GetServer(ctx context.Context, name string) (*hcloud.Server, error) {
	server, _, err := c.hcloud.Server.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch server %s: %w", name, err)
	}
	return server, nil
}

// CreateServer creates a new server
func (c *Client) CreateServer(ctx context.Context, opts hcloud.ServerCreateOpts) (*hcloud.Server, error) {
	result, _, err := c.hcloud.Server.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	// Wait for the action to complete
	if result.Action != nil {
		if err := c.waitForAction(ctx, result.Action); err != nil {
			return nil, fmt.Errorf("server creation action failed: %w", err)
		}
	}

	// Wait for any next actions
	if len(result.NextActions) > 0 {
		if err := c.waitForActions(ctx, result.NextActions); err != nil {
			return nil, fmt.Errorf("server creation next actions failed: %w", err)
		}
	}

	return result.Server, nil
}

// DeleteServer deletes a server
func (c *Client) DeleteServer(ctx context.Context, server *hcloud.Server) error {
	result, _, err := c.hcloud.Server.DeleteWithResult(ctx, server)
	if err != nil {
		return fmt.Errorf("failed to delete server %s: %w", server.Name, err)
	}

	// Wait for the delete action to complete
	if result.Action != nil {
		if err := c.waitForAction(ctx, result.Action); err != nil {
			return fmt.Errorf("server deletion action failed: %w", err)
		}
	}

	return nil
}

// CreateNetwork creates a new network
func (c *Client) CreateNetwork(ctx context.Context, opts hcloud.NetworkCreateOpts) (*hcloud.Network, error) {
	network, _, err := c.hcloud.Network.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}
	return network, nil
}

// GetNetwork returns a specific network by name
func (c *Client) GetNetwork(ctx context.Context, name string) (*hcloud.Network, error) {
	network, _, err := c.hcloud.Network.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch network %s: %w", name, err)
	}
	return network, nil
}

// AddRouteToNetwork adds a route to a network
func (c *Client) AddRouteToNetwork(ctx context.Context, network *hcloud.Network, opts hcloud.NetworkAddRouteOpts) error {
	action, _, err := c.hcloud.Network.AddRoute(ctx, network, opts)
	if err != nil {
		return fmt.Errorf("failed to add route to network %s: %w", network.Name, err)
	}

	// Wait for the action to complete
	if err := c.waitForAction(ctx, action); err != nil {
		return fmt.Errorf("network add route action failed: %w", err)
	}

	return nil
}

// DeleteNetwork deletes a network
func (c *Client) DeleteNetwork(ctx context.Context, network *hcloud.Network) error {
	_, err := c.hcloud.Network.Delete(ctx, network)
	if err != nil {
		return fmt.Errorf("failed to delete network %s: %w", network.Name, err)
	}

	// Wait for network to actually be deleted
	// Poll with timeout to prevent infinite loops on persistent API issues
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(2 * time.Minute)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("timeout waiting for network %s to be deleted", network.Name)
		case <-ticker.C:
			// Check if network still exists
			net, _, err := c.hcloud.Network.GetByID(ctx, network.ID)
			if err != nil {
				// Only treat 'not found' errors as successful deletion
				if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
					return nil
				}
				// For other errors (e.g., transient network issues, API throttling),
				// continue polling rather than failing the deletion immediately.
				// The timeout will prevent infinite loops on persistent API issues.
			}
			if net == nil {
				// Network is deleted
				return nil
			}
		}
	}
}

// CreateSSHKey creates a new SSH key
func (c *Client) CreateSSHKey(ctx context.Context, opts hcloud.SSHKeyCreateOpts) (*hcloud.SSHKey, error) {
	sshKey, _, err := c.hcloud.SSHKey.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH key: %w", err)
	}
	return sshKey, nil
}

// GetSSHKey returns a specific SSH key by name
func (c *Client) GetSSHKey(ctx context.Context, name string) (*hcloud.SSHKey, error) {
	sshKey, _, err := c.hcloud.SSHKey.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch SSH key %s: %w", name, err)
	}
	return sshKey, nil
}

// DeleteSSHKey deletes an SSH key
func (c *Client) DeleteSSHKey(ctx context.Context, sshKey *hcloud.SSHKey) error {
	_, err := c.hcloud.SSHKey.Delete(ctx, sshKey)
	if err != nil {
		return fmt.Errorf("failed to delete SSH key %s: %w", sshKey.Name, err)
	}
	return nil
}

// CreateFirewall creates a new firewall
func (c *Client) CreateFirewall(ctx context.Context, opts hcloud.FirewallCreateOpts) (*hcloud.Firewall, error) {
	result, _, err := c.hcloud.Firewall.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create firewall: %w", err)
	}

	// Wait for any actions to complete
	if len(result.Actions) > 0 {
		if err := c.waitForActions(ctx, result.Actions); err != nil {
			return nil, fmt.Errorf("firewall creation action failed: %w", err)
		}
	}

	return result.Firewall, nil
}

// GetFirewall returns a specific firewall by name
func (c *Client) GetFirewall(ctx context.Context, name string) (*hcloud.Firewall, error) {
	firewall, _, err := c.hcloud.Firewall.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch firewall %s: %w", name, err)
	}
	return firewall, nil
}

// DeleteFirewall deletes a firewall
func (c *Client) DeleteFirewall(ctx context.Context, firewall *hcloud.Firewall) error {
	_, err := c.hcloud.Firewall.Delete(ctx, firewall)
	if err != nil {
		return fmt.Errorf("failed to delete firewall %s: %w", firewall.Name, err)
	}

	// Wait for firewall to actually be deleted
	// Poll with timeout to prevent infinite loops on persistent API issues
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(2 * time.Minute)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("timeout waiting for firewall %s to be deleted", firewall.Name)
		case <-ticker.C:
			// Check if firewall still exists
			fw, _, err := c.hcloud.Firewall.GetByID(ctx, firewall.ID)
			if err != nil {
				// Only treat 'not found' errors as successful deletion
				if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
					return nil
				}
				// For other errors (e.g., transient network issues, API throttling),
				// continue polling rather than failing the deletion immediately.
				// The timeout will prevent infinite loops on persistent API issues.
			}
			if fw == nil {
				// Firewall is deleted
				return nil
			}
		}
	}
}

// CreateLoadBalancer creates a new load balancer
func (c *Client) CreateLoadBalancer(ctx context.Context, opts hcloud.LoadBalancerCreateOpts) (*hcloud.LoadBalancer, error) {
	result, _, err := c.hcloud.LoadBalancer.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create load balancer: %w", err)
	}

	// Wait for the action to complete
	if err := c.waitForAction(ctx, result.Action); err != nil {
		return nil, fmt.Errorf("load balancer creation action failed: %w", err)
	}

	return result.LoadBalancer, nil
}

// GetLoadBalancer returns a specific load balancer by name
func (c *Client) GetLoadBalancer(ctx context.Context, name string) (*hcloud.LoadBalancer, error) {
	lb, _, err := c.hcloud.LoadBalancer.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch load balancer %s: %w", name, err)
	}
	return lb, nil
}

// ListLoadBalancers returns all load balancers matching the provided options, including label selectors
func (c *Client) ListLoadBalancers(ctx context.Context, opts hcloud.LoadBalancerListOpts) ([]*hcloud.LoadBalancer, error) {
	lbs, err := c.hcloud.LoadBalancer.AllWithOpts(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list load balancers: %w", err)
	}
	return lbs, nil
}

// DeleteLoadBalancer deletes a load balancer
func (c *Client) DeleteLoadBalancer(ctx context.Context, lb *hcloud.LoadBalancer) error {
	_, err := c.hcloud.LoadBalancer.Delete(ctx, lb)
	if err != nil {
		return fmt.Errorf("failed to delete load balancer %s: %w", lb.Name, err)
	}

	// Wait for load balancer to actually be deleted
	// Poll with timeout to prevent infinite loops on persistent API issues
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(2 * time.Minute)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("timeout waiting for load balancer %s to be deleted", lb.Name)
		case <-ticker.C:
			// Check if load balancer still exists
			loadBalancer, _, err := c.hcloud.LoadBalancer.GetByID(ctx, lb.ID)
			if err != nil {
				// Only treat 'not found' errors as successful deletion
				if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
					return nil
				}
				// For other errors (e.g., transient network issues, API throttling),
				// continue polling rather than failing the deletion immediately.
				// The timeout will prevent infinite loops on persistent API issues.
			}
			if loadBalancer == nil {
				// Load balancer is deleted
				return nil
			}
		}
	}
}

// AddServiceToLoadBalancer adds a service to an existing load balancer
func (c *Client) AddServiceToLoadBalancer(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAddServiceOpts) error {
	action, _, err := c.hcloud.LoadBalancer.AddService(ctx, lb, opts)
	if err != nil {
		return fmt.Errorf("failed to add service to load balancer %s: %w", lb.Name, err)
	}

	// Wait for the action to complete
	if err := c.waitForAction(ctx, action); err != nil {
		return fmt.Errorf("add service action failed: %w", err)
	}

	return nil
}

// AddLabelSelectorTargetToLoadBalancer adds a label selector target to an existing load balancer
func (c *Client) AddLabelSelectorTargetToLoadBalancer(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAddLabelSelectorTargetOpts) error {
	action, _, err := c.hcloud.LoadBalancer.AddLabelSelectorTarget(ctx, lb, opts)
	if err != nil {
		return fmt.Errorf("failed to add label selector target to load balancer %s: %w", lb.Name, err)
	}

	// Wait for the action to complete
	if err := c.waitForAction(ctx, action); err != nil {
		return fmt.Errorf("add label selector target action failed: %w", err)
	}

	return nil
}

// AddServerTargetToLoadBalancer adds a server target to an existing load balancer
func (c *Client) AddServerTargetToLoadBalancer(ctx context.Context, lb *hcloud.LoadBalancer, opts hcloud.LoadBalancerAddServerTargetOpts) error {
	action, _, err := c.hcloud.LoadBalancer.AddServerTarget(ctx, lb, opts)
	if err != nil {
		return fmt.Errorf("failed to add server target to load balancer %s: %w", lb.Name, err)
	}

	// Wait for the action to complete
	if err := c.waitForAction(ctx, action); err != nil {
		return fmt.Errorf("add server target action failed: %w", err)
	}

	return nil
}

// waitForAction waits for a single action to complete
func (c *Client) waitForAction(ctx context.Context, action *hcloud.Action) error {
	if action == nil {
		return nil
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			a, _, err := c.hcloud.Action.GetByID(ctx, action.ID)
			if err != nil {
				return fmt.Errorf("failed to get action status: %w", err)
			}

			if a.Status == hcloud.ActionStatusSuccess {
				return nil
			}
			if a.Status == hcloud.ActionStatusError {
				return fmt.Errorf("action failed: %s", a.ErrorMessage)
			}
		}
	}
}

// waitForActions waits for multiple actions to complete
func (c *Client) waitForActions(ctx context.Context, actions []*hcloud.Action) error {
	for _, action := range actions {
		if err := c.waitForAction(ctx, action); err != nil {
			return err
		}
	}
	return nil
}

// WaitForServerStatus waits for a server to reach the specified status
func (c *Client) WaitForServerStatus(ctx context.Context, server *hcloud.Server, targetStatus hcloud.ServerStatus, timeout time.Duration) error {
	if server == nil {
		return fmt.Errorf("server is nil")
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeoutTimer.C:
			return fmt.Errorf("timeout waiting for server %s to reach status %s", server.Name, targetStatus)
		case <-ticker.C:
			// Refresh server status
			srv, _, err := c.hcloud.Server.GetByID(ctx, server.ID)
			if err != nil {
				return fmt.Errorf("failed to get server status: %w", err)
			}
			if srv == nil {
				return fmt.Errorf("server %s not found", server.Name)
			}

			if srv.Status == targetStatus {
				return nil
			}
		}
	}
}

// GetHCloudClient returns the underlying hcloud client for advanced operations
func (c *Client) GetHCloudClient() *hcloud.Client {
	return c.hcloud
}

// GetZone returns a specific DNS zone by name
func (c *Client) GetZone(ctx context.Context, name string) (*hcloud.Zone, error) {
	zone, _, err := c.hcloud.Zone.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch zone %s: %w", name, err)
	}
	return zone, nil
}

// CreateZone creates a new DNS zone
func (c *Client) CreateZone(ctx context.Context, opts hcloud.ZoneCreateOpts) (*hcloud.Zone, error) {
	result, _, err := c.hcloud.Zone.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create zone: %w", err)
	}

	// Wait for the action to complete if present
	if result.Action != nil {
		if err := c.waitForAction(ctx, result.Action); err != nil {
			return nil, fmt.Errorf("zone creation action failed: %w", err)
		}
	}

	return result.Zone, nil
}

// DeleteZone deletes a DNS zone
func (c *Client) DeleteZone(ctx context.Context, zone *hcloud.Zone) error {
	result, _, err := c.hcloud.Zone.Delete(ctx, zone)
	if err != nil {
		return fmt.Errorf("failed to delete zone %s: %w", zone.Name, err)
	}

	// Wait for the action to complete if present
	if result.Action != nil {
		if err := c.waitForAction(ctx, result.Action); err != nil {
			return fmt.Errorf("zone deletion action failed: %w", err)
		}
	}

	// Wait for zone to actually be deleted
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(2 * time.Minute)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("timeout waiting for zone %s to be deleted", zone.Name)
		case <-ticker.C:
			// Check if zone still exists
			z, _, err := c.hcloud.Zone.GetByID(ctx, zone.ID)
			if err != nil {
				// Only treat 'not found' errors as successful deletion
				if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
					return nil
				}
				// For other errors, continue polling
			}
			if z == nil {
				// Zone is deleted
				return nil
			}
		}
	}
}

// GetZoneRRSet returns a specific DNS record set by zone and name
func (c *Client) GetZoneRRSet(ctx context.Context, zone *hcloud.Zone, name string, rrType hcloud.ZoneRRSetType) (*hcloud.ZoneRRSet, error) {
	// List all RRSets and filter by name and type
	rrsets, err := c.hcloud.Zone.AllRRSetsWithOpts(ctx, zone, hcloud.ZoneRRSetListOpts{
		Name: name,
		Type: []hcloud.ZoneRRSetType{rrType},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list zone RRSets: %w", err)
	}

	if len(rrsets) == 0 {
		return nil, nil
	}

	// Return the first match. In practice, the combination of zone + name + type
	// should uniquely identify a single RRSet. If multiple matches exist, this
	// indicates an unexpected state in the DNS zone configuration.
	return rrsets[0], nil
}

// CreateZoneRRSet creates a new DNS record set
func (c *Client) CreateZoneRRSet(ctx context.Context, zone *hcloud.Zone, opts hcloud.ZoneRRSetCreateOpts) (*hcloud.ZoneRRSet, error) {
	result, _, err := c.hcloud.Zone.CreateRRSet(ctx, zone, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create zone RRSet: %w", err)
	}

	// Wait for the action to complete if present
	if result.Action != nil {
		if err := c.waitForAction(ctx, result.Action); err != nil {
			return nil, fmt.Errorf("zone RRSet creation action failed: %w", err)
		}
	}

	return result.RRSet, nil
}

// DeleteZoneRRSet deletes a DNS record set
func (c *Client) DeleteZoneRRSet(ctx context.Context, rrset *hcloud.ZoneRRSet) error {
	result, _, err := c.hcloud.Zone.DeleteRRSet(ctx, rrset)
	if err != nil {
		return fmt.Errorf("failed to delete zone RRSet: %w", err)
	}

	// Wait for the action to complete if present
	if result.Action != nil {
		if err := c.waitForAction(ctx, result.Action); err != nil {
			return fmt.Errorf("zone RRSet deletion action failed: %w", err)
		}
	}

	return nil
}

// CreateManagedCertificate creates a new managed SSL certificate
// Managed certificates are automatically validated using DNS records
func (c *Client) CreateManagedCertificate(ctx context.Context, opts hcloud.CertificateCreateOpts) (*hcloud.Certificate, error) {
	result, _, err := c.hcloud.Certificate.CreateCertificate(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Note: We don't wait for certificate to be issued as it can take up to 5 minutes
	// Hetzner will issue it in the background using DNS validation
	return result.Certificate, nil
}

// GetCertificate returns a specific certificate by name
func (c *Client) GetCertificate(ctx context.Context, name string) (*hcloud.Certificate, error) {
	cert, _, err := c.hcloud.Certificate.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch certificate %s: %w", name, err)
	}
	return cert, nil
}

// DeleteCertificate deletes a certificate
func (c *Client) DeleteCertificate(ctx context.Context, cert *hcloud.Certificate) error {
	_, err := c.hcloud.Certificate.Delete(ctx, cert)
	if err != nil {
		return fmt.Errorf("failed to delete certificate %s: %w", cert.Name, err)
	}

	// Wait for certificate to actually be deleted
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(2 * time.Minute)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("timeout waiting for certificate %s to be deleted", cert.Name)
		case <-ticker.C:
			// Check if certificate still exists
			certificate, _, err := c.hcloud.Certificate.GetByID(ctx, cert.ID)
			if err != nil {
				// Only treat 'not found' errors as successful deletion
				if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
					return nil
				}
				// For other errors (e.g., transient network issues, API throttling),
				// continue polling rather than failing the deletion immediately.
				// The timeout will prevent infinite loops on persistent API issues.
			}
			if certificate == nil {
				// Certificate is deleted
				return nil
			}
		}
	}
}

// CreatePlacementGroup creates a new placement group
func (c *Client) CreatePlacementGroup(ctx context.Context, opts hcloud.PlacementGroupCreateOpts) (*hcloud.PlacementGroup, error) {
	result, _, err := c.hcloud.PlacementGroup.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create placement group: %w", err)
	}

	// Wait for the action to complete if present
	if result.Action != nil {
		if err := c.waitForAction(ctx, result.Action); err != nil {
			return nil, fmt.Errorf("placement group creation action failed: %w", err)
		}
	}

	return result.PlacementGroup, nil
}

// GetPlacementGroup returns a specific placement group by name
func (c *Client) GetPlacementGroup(ctx context.Context, name string) (*hcloud.PlacementGroup, error) {
	pg, _, err := c.hcloud.PlacementGroup.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch placement group %s: %w", name, err)
	}
	return pg, nil
}

// ListPlacementGroups returns all placement groups matching the label selector
func (c *Client) ListPlacementGroups(ctx context.Context, opts hcloud.PlacementGroupListOpts) ([]*hcloud.PlacementGroup, error) {
	pgs, err := c.hcloud.PlacementGroup.AllWithOpts(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list placement groups: %w", err)
	}
	return pgs, nil
}

// ChangeServerProtection changes the protection level of a server.
// When delete is true, the server is protected from deletion.
func (c *Client) ChangeServerProtection(ctx context.Context, server *hcloud.Server, delete bool) error {
	action, _, err := c.hcloud.Server.ChangeProtection(ctx, server, hcloud.ServerChangeProtectionOpts{
		Delete: hcloud.Ptr(delete),
	})
	if err != nil {
		return fmt.Errorf("failed to change protection for server %s: %w", server.Name, err)
	}
	return c.waitForAction(ctx, action)
}

// ChangeLoadBalancerProtection changes the protection level of a load balancer.
// When delete is true, the load balancer is protected from deletion.
func (c *Client) ChangeLoadBalancerProtection(ctx context.Context, lb *hcloud.LoadBalancer, delete bool) error {
	action, _, err := c.hcloud.LoadBalancer.ChangeProtection(ctx, lb, hcloud.LoadBalancerChangeProtectionOpts{
		Delete: hcloud.Ptr(delete),
	})
	if err != nil {
		return fmt.Errorf("failed to change protection for load balancer %s: %w", lb.Name, err)
	}
	return c.waitForAction(ctx, action)
}

// ChangeNetworkProtection changes the protection level of a network.
// When delete is true, the network is protected from deletion.
func (c *Client) ChangeNetworkProtection(ctx context.Context, network *hcloud.Network, delete bool) error {
	action, _, err := c.hcloud.Network.ChangeProtection(ctx, network, hcloud.NetworkChangeProtectionOpts{
		Delete: hcloud.Ptr(delete),
	})
	if err != nil {
		return fmt.Errorf("failed to change protection for network %s: %w", network.Name, err)
	}
	return c.waitForAction(ctx, action)
}

// DeletePlacementGroup deletes a placement group
func (c *Client) DeletePlacementGroup(ctx context.Context, pg *hcloud.PlacementGroup) error {
	_, err := c.hcloud.PlacementGroup.Delete(ctx, pg)
	if err != nil {
		return fmt.Errorf("failed to delete placement group %s: %w", pg.Name, err)
	}

	// Wait for placement group to actually be deleted
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(2 * time.Minute)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("timeout waiting for placement group %s to be deleted", pg.Name)
		case <-ticker.C:
			// Check if placement group still exists
			placementGroup, _, err := c.hcloud.PlacementGroup.GetByID(ctx, pg.ID)
			if err != nil {
				// Only treat 'not found' errors as successful deletion
				if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
					return nil
				}
				// For other errors, continue polling
			}
			if placementGroup == nil {
				// Placement group is deleted
				return nil
			}
		}
	}
}
