package cluster

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/internal/addons"
	"github.com/magenx/kuberaptor/internal/cloudinit"
	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/internal/util"
	"github.com/magenx/kuberaptor/pkg/hetzner"
	"github.com/magenx/kuberaptor/pkg/k3s"
)

const (
	// k3sKubeconfigPath is the default path to the k3s kubeconfig file
	// This constant is for documentation; actual commands use hardcoded paths for security
	k3sKubeconfigPath = "/etc/rancher/k3s/k3s.yaml"
	// k3sKubeconfigCheckCmd is the command to check if kubeconfig exists
	k3sKubeconfigCheckCmd = "test -f /etc/rancher/k3s/k3s.yaml && echo 'exists'"
	// k3sKubeconfigReadCmd is the command to read the kubeconfig file
	k3sKubeconfigReadCmd = "sudo cat /etc/rancher/k3s/k3s.yaml"
	// k3sNodeTokenPath is the path to the k3s node-token file on the server
	k3sNodeTokenPath = "/var/lib/rancher/k3s/server/node-token"
)

// CreatorEnhanced handles cluster creation with full implementation
type CreatorEnhanced struct {
	Config           *config.Main
	HetznerClient    *hetzner.Client
	SSHClient        *util.SSH
	ctx              context.Context
	k3sToken         string
	staticPools      []config.WorkerNodePool
	autoscalingPools []config.WorkerNodePool
}

// NewCreatorEnhanced creates a new enhanced cluster creator
func NewCreatorEnhanced(cfg *config.Main, hetznerClient *hetzner.Client) (*CreatorEnhanced, error) {
	// Get SSH keys (either from paths or inline content)
	privKey, err := cfg.Networking.SSH.GetPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	pubKey, err := cfg.Networking.SSH.GetPublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	sshClient := util.NewSSHFromKeys(privKey, pubKey)

	// Generate k3s token
	token, err := k3s.GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate k3s token: %w", err)
	}

	// Separate static and autoscaling worker pools
	staticPools, autoscalingPools := separateWorkerPools(cfg.WorkerNodePools)

	return &CreatorEnhanced{
		Config:           cfg,
		HetznerClient:    hetznerClient,
		SSHClient:        sshClient,
		ctx:              context.Background(),
		k3sToken:         token,
		staticPools:      staticPools,
		autoscalingPools: autoscalingPools,
	}, nil
}

// Run executes the cluster creation process
func (c *CreatorEnhanced) Run() error {
	util.LogInfo("Starting cluster creation", c.Config.ClusterName)
	util.LogInfo(fmt.Sprintf("K3s token: %s", c.k3sToken[:16]), c.Config.ClusterName)

	// Step 1: Create SSH key in Hetzner
	util.LogInfo("Creating SSH key", "ssh key")
	sshKey, err := c.createSSHKey()
	if err != nil {
		util.LogError(fmt.Sprintf("Failed to create SSH key: %v", err), "ssh key")
		return fmt.Errorf("failed to create SSH key: %w", err)
	}
	util.LogSuccess(fmt.Sprintf("SSH key created: %s", sshKey.Name), "ssh key")

	// Step 2: Create network if private network is enabled
	var network *hcloud.Network
	var natGateways []*hcloud.Server // Support multiple NAT gateways (one per location)
	if c.Config.Networking.PrivateNetwork.Enabled {
		util.LogInfo("Creating private network", "network")
		network, err = c.createNetwork()
		if err != nil {
			util.LogError(fmt.Sprintf("Failed to create network: %v", err), "network")
			return fmt.Errorf("failed to create network: %w", err)
		}
		util.LogSuccess(fmt.Sprintf("Network created: %s", network.Name), "network")

		// Step 2.1: Create NAT gateway(s) if enabled
		if c.Config.Networking.PrivateNetwork.NATGateway != nil && c.Config.Networking.PrivateNetwork.NATGateway.Enabled {
			// Determine locations for NAT gateways
			natLocations := c.Config.Networking.PrivateNetwork.NATGateway.Locations
			if len(natLocations) == 0 {
				// If no locations specified, use master locations for full coverage
				natLocations = c.Config.MastersPool.Locations
			}

			util.LogInfo(fmt.Sprintf("Creating %d NAT gateway(s) across locations", len(natLocations)), "nat gateway")
			natGateways, err = c.createNATGateways(sshKey, network, natLocations)
			if err != nil {
				return fmt.Errorf("failed to create NAT gateways: %w", err)
			}
			util.LogSuccess(fmt.Sprintf("Created %d NAT gateway(s)", len(natGateways)), "nat gateway")

			// Wait for NAT gateways to be ready
			spinner := util.NewSpinner("Waiting for NAT gateway(s) to be ready", "nat gateway")
			spinner.Start()
			if err := c.waitForNodes(natGateways); err != nil {
				spinner.Stop(true)
				return fmt.Errorf("failed waiting for NAT gateways: %w", err)
			}
			spinner.Stop(true)
			util.LogSuccess("NAT gateway(s) are ready", "nat gateway")

			// Step 2.2: Add routes to network via NAT gateways
			util.LogInfo("Adding default routes via NAT gateways to private network", "nat gateway")
			if err := c.addNATGatewayRoutes(network, natGateways); err != nil {
				return fmt.Errorf("failed to add NAT gateway routes: %w", err)
			}
			util.LogSuccess("Default routes via NAT gateways added to private network", "nat gateway")

			// Step 2.3: Configure SSH to use first NAT gateway as bastion host
			bastionIP, err := GetServerPublicIP(natGateways[0])
			if err != nil {
				return fmt.Errorf("failed to get NAT gateway public IP: %w", err)
			}
			c.SSHClient.SetBastion(bastionIP, c.Config.Networking.SSH.Port)
			util.LogInfo(fmt.Sprintf("Using NAT gateway %s as SSH bastion host", bastionIP), "nat gateway")
		}
	}

	// Step 3: Create master nodes
	util.LogInfo(fmt.Sprintf("Creating %d master node(s)", c.Config.MastersPool.InstanceCount), "master")
	masters, err := c.createMasterNodes(sshKey, network)
	if err != nil {
		util.LogError(fmt.Sprintf("Failed to create master nodes: %v", err), "master")
		return fmt.Errorf("failed to create master nodes: %w", err)
	}
	util.LogSuccess(fmt.Sprintf("Created %d master node(s)", len(masters)), "master")

	// Step 4: Create firewall for cluster
	util.LogInfo("Creating firewall", "firewall")
	if err := c.createFirewall(network, masters); err != nil {
		return fmt.Errorf("failed to create firewall: %w", err)
	}

	// Step 5: Wait for masters to be ready
	spinner := util.NewSpinner("Waiting for master nodes to be ready", "master")
	spinner.Start()
	if err := c.waitForNodes(masters); err != nil {
		spinner.Stop(true)
		return fmt.Errorf("failed waiting for masters: %w", err)
	}
	spinner.Stop(true)
	util.LogSuccess("Master nodes are ready", "master")

	// Step 5a: Create API load balancer(s) BEFORE installing k3s (if configured)
	// This ensures the load balancer IP can be included in the TLS SANs
	var apiLoadBalancers []*hcloud.LoadBalancer
	if c.Config.APILoadBalancer.Enabled {
		networkMgr := NewNetworkResourceManager(c.Config, c.HetznerClient)

		// Create one API load balancer per location for regional autonomy
		util.LogInfo(fmt.Sprintf("Creating %d API load balancer(s) for Kubernetes API", len(c.Config.MastersPool.Locations)), "load balancer")
		lbs, err := networkMgr.CreateAPILoadBalancers(masters, c.Config.MastersPool.Locations, network)
		if err != nil {
			return fmt.Errorf("failed to create API load balancers: %w", err)
		}
		apiLoadBalancers = lbs
	}

	// Step 6: Install k3s on all masters in parallel
	spinner = util.NewSpinner(fmt.Sprintf("Installing k3s on %d master node(s)", len(masters)), "master")
	spinner.Start()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for i := 0; i < len(masters); i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			if err := c.installK3sOnMaster(masters[index], masters, apiLoadBalancers); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("failed to install k3s on master %d: %w", index+1, err))
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	hasErrors := len(errors) > 0
	spinner.Stop(hasErrors)

	if hasErrors {
		util.LogError(fmt.Sprintf("Errors installing k3s on masters: %v", errors), "master")
		return fmt.Errorf("errors installing k3s on masters: %v", errors)
	}

	util.LogSuccess(fmt.Sprintf("K3s installed on %d master node(s)", len(masters)), "master")

	// Step 7: Create worker nodes (if configured)
	// Only create nodes for static (non-autoscaling) pools.
	// Autoscaling pools will be managed entirely by the cluster autoscaler,
	// which will create nodes from 0 up to max_instances as needed.
	totalWorkers := 0
	for _, pool := range c.staticPools {
		totalWorkers += pool.InstanceCount
	}

	if len(c.staticPools) > 0 && totalWorkers > 0 {
		util.LogInfo(fmt.Sprintf("Creating %d worker node(s) across %d static pool(s)", totalWorkers, len(c.staticPools)), "worker")
		workers, workerToPool, err := c.createWorkerNodesFromPools(sshKey, network, c.staticPools)
		if err != nil {
			return fmt.Errorf("failed to create worker nodes: %w", err)
		}

		// Step 8: Wait for workers and install k3s
		if len(workers) > 0 {
			spinner := util.NewSpinner("Waiting for worker nodes to be ready", "worker")
			spinner.Start()
			if err := c.waitForNodes(workers); err != nil {
				spinner.Stop(true)
				return fmt.Errorf("failed waiting for workers: %w", err)
			}
			spinner.Stop(true)
			util.LogSuccess("Worker nodes are ready", "worker")

			// Ensure we have at least one master before retrieving node-token
			if len(masters) == 0 {
				return fmt.Errorf("no master nodes available to retrieve node-token")
			}

			// Retrieve node-token from master once before installing on all workers
			// This avoids retrieving it multiple times for each worker
			nodeToken, err := c.getNodeTokenFromMaster(masters[0])
			if err != nil {
				return fmt.Errorf("failed to get node-token from master: %w", err)
			}

			spinner = util.NewSpinner("Installing k3s on worker nodes", "worker")
			spinner.Start()

			var wg sync.WaitGroup
			var mu sync.Mutex
			var errors []error

			for _, worker := range workers {
				wg.Add(1)
				go func(w *hcloud.Server) {
					defer wg.Done()
					// Get pool configuration for this worker
					pool, ok := workerToPool[w.Name]
					if !ok {
						mu.Lock()
						errors = append(errors, fmt.Errorf("pool configuration not found for worker %s", w.Name))
						mu.Unlock()
						return
					}
					if err := c.installK3sOnWorkerWithToken(w, masters[0], &pool, nodeToken); err != nil {
						mu.Lock()
						errors = append(errors, fmt.Errorf("failed to install k3s on worker %s: %w", w.Name, err))
						mu.Unlock()
					}
				}(worker)
			}

			wg.Wait()
			hasErrors := len(errors) > 0
			spinner.Stop(hasErrors)

			if hasErrors {
				return fmt.Errorf("errors installing k3s on workers: %v", errors)
			}

			util.LogSuccess("K3s installed on all worker nodes", "worker")
		}
	}

	// Step 9: Create global load balancer(s) (if enabled)
	if c.Config.LoadBalancer.Enabled {
		networkMgr := NewNetworkResourceManager(c.Config, c.HetznerClient)

		// Step 9a: Create DNS zone (if enabled) - must be created before SSL certificate
		if c.Config.DNSZone.Enabled && c.Config.Domain != "" {
			util.LogInfo("Creating DNS zone for domain", "dns")
			_, err := networkMgr.CreateDNSZone()
			if err != nil {
				return fmt.Errorf("failed to create DNS zone: %w", err)
			}
		}

		// Step 9b: Create SSL certificate (if enabled) - must be created before load balancer
		var certificate *hcloud.Certificate
		if c.Config.SSLCertificate.Enabled {
			util.LogInfo("Creating SSL certificate", "ssl")
			cert, err := networkMgr.CreateSSLCertificate()
			if err != nil {
				return fmt.Errorf("failed to create SSL certificate: %w", err)
			}
			certificate = cert
		}

		// Step 9c: Create global load balancer(s) with certificate attached
		// Determine locations for load balancers
		lbLocations := c.Config.LoadBalancer.Locations
		if len(lbLocations) == 0 {
			// If no locations specified, use first master location (single LB)
			lbLocations = []string{c.Config.MastersPool.Locations[0]}
		}

		util.LogInfo(fmt.Sprintf("Creating %d global load balancer(s) for application traffic", len(lbLocations)), "load balancer")
		_, err := networkMgr.CreateGlobalLoadBalancers(network, lbLocations, certificate)
		if err != nil {
			return fmt.Errorf("failed to create global load balancers: %w", err)
		}
	}

	// Step 10: Retrieve kubeconfig
	util.LogInfo("Retrieving kubeconfig", "kubeconfig")
	if err := c.retrieveKubeconfig(masters[0], apiLoadBalancers); err != nil {
		return fmt.Errorf("failed to retrieve kubeconfig: %w", err)
	}
	util.LogSuccess(fmt.Sprintf("Kubeconfig saved to: %s", c.Config.KubeconfigPath), "kubeconfig")

	// Step 11: Install addons
	// Use pre-computed autoscaling pools (already separated during initialization)
	if err := c.installAddons(masters[0], masters, c.autoscalingPools); err != nil {
		return fmt.Errorf("failed to install addons: %w", err)
	}

	fmt.Println()
	util.LogSuccess("Cluster creation completed successfully!", c.Config.ClusterName)
	fmt.Println()

	return nil
}

// createSSHKey creates and uploads SSH key to Hetzner
func (c *CreatorEnhanced) createSSHKey() (*hcloud.SSHKey, error) {
	// Get public key content (from file or inline)
	pubKeyBytes, err := c.SSHClient.GetPublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}
	pubKeyContent := string(pubKeyBytes)

	keyName := fmt.Sprintf("%s-ssh-key", c.Config.ClusterName)

	// Check if key already exists
	existingKey, err := c.HetznerClient.GetSSHKey(c.ctx, keyName)
	if err == nil && existingKey != nil {
		util.LogInfo("SSH key already exists, using existing key", "ssh key")
		return existingKey, nil
	}

	// Create SSH key
	return c.HetznerClient.CreateSSHKey(c.ctx, hcloud.SSHKeyCreateOpts{
		Name:      keyName,
		PublicKey: pubKeyContent,
	})
}

// createNetwork creates a private network
func (c *CreatorEnhanced) createNetwork() (*hcloud.Network, error) {
	// Use cluster name as network name (no "-network" suffix)
	networkName := c.Config.ClusterName

	// Check if network already exists
	existingNetwork, err := c.HetznerClient.GetNetwork(c.ctx, networkName)
	if err == nil && existingNetwork != nil {
		util.LogInfo("Network already exists, using existing network", "network")
		return existingNetwork, nil
	}

	// Parse subnet
	_, ipNet, err := net.ParseCIDR(c.Config.Networking.PrivateNetwork.Subnet)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet: %w", err)
	}

	// Create network
	return c.HetznerClient.CreateNetwork(c.ctx, hcloud.NetworkCreateOpts{
		Name:    networkName,
		IPRange: (*net.IPNet)(ipNet),
		Subnets: []hcloud.NetworkSubnet{
			{
				Type:        hcloud.NetworkSubnetTypeCloud,
				IPRange:     (*net.IPNet)(ipNet),
				NetworkZone: hcloud.NetworkZoneEUCentral,
			},
		},
	})
}

// createNATGateways creates multiple NAT gateway instances for multi-location deployment
// Each location gets its own NAT gateway for regional outbound internet access
func (c *CreatorEnhanced) createNATGateways(sshKey *hcloud.SSHKey, network *hcloud.Network, locations []string) ([]*hcloud.Server, error) {
	gateways := make([]*hcloud.Server, 0, len(locations))

	// Generate cloud-init for NAT gateway once (same for all)
	cloudInitData, err := c.generateNATGatewayCloudInit()
	if err != nil {
		return nil, fmt.Errorf("failed to generate cloud-init for NAT gateway: %w", err)
	}

	// Get server type once (same for all)
	instanceType := c.Config.Networking.PrivateNetwork.NATGateway.InstanceType
	serverType, err := c.HetznerClient.GetServerType(c.ctx, instanceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get server type: %w", err)
	}

	// Get image once (same for all)
	image, err := c.HetznerClient.GetImage(c.ctx, c.Config.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}

	// Create one NAT gateway per location
	for _, location := range locations {
		nodeName := fmt.Sprintf("%s-nat-gateway-%s", c.Config.ClusterName, location)

		// Check if NAT gateway already exists for this location
		existingServer, err := c.HetznerClient.GetServer(c.ctx, nodeName)
		if err == nil && existingServer != nil {
			util.LogInfo(fmt.Sprintf("NAT gateway already exists for %s, using existing instance", location), "nat gateway")
			gateways = append(gateways, existingServer)
			continue
		}

		// Get location
		loc, err := c.HetznerClient.GetLocation(c.ctx, location)
		if err != nil {
			return nil, fmt.Errorf("failed to get location %s: %w", location, err)
		}

		// Prepare server options
		opts := hcloud.ServerCreateOpts{
			Name:       nodeName,
			ServerType: serverType,
			Image:      image,
			Location:   loc,
			SSHKeys:    []*hcloud.SSHKey{sshKey},
			UserData:   cloudInitData,
			Labels:     c.buildNATGatewayHetznerServerLabels(location),
			Networks:   []*hcloud.Network{network},
			PublicNet: &hcloud.ServerCreatePublicNet{
				EnableIPv4: true,
				EnableIPv6: false,
			},
		}

		// Create server
		util.LogInfo(fmt.Sprintf("Creating NAT gateway: %s in %s", nodeName, location), "nat gateway")
		server, err := c.HetznerClient.CreateServer(c.ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create NAT gateway server %s: %w", nodeName, err)
		}

		gateways = append(gateways, server)
	}

	return gateways, nil
}

var (
	// defaultRouteDestination is the CIDR for all traffic (default route)
	defaultRouteDestination = mustParseCIDR("0.0.0.0/0")
)

// mustParseCIDR parses a CIDR string and panics if it fails
// Only used for compile-time constant values that are known to be valid
func mustParseCIDR(cidr string) *net.IPNet {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(fmt.Sprintf("invalid CIDR constant: %s", cidr))
	}
	return ipNet
}

// addNATGatewayRoutes adds default routes to the network via multiple NAT gateways
// For multi-location deployments, this creates location-specific routing
func (c *CreatorEnhanced) addNATGatewayRoutes(network *hcloud.Network, natGateways []*hcloud.Server) error {
	// For simplicity, use the first NAT gateway as the default route
	// In production, you might want more sophisticated routing (e.g., per-subnet routes)
	if len(natGateways) == 0 {
		return fmt.Errorf("no NAT gateways provided")
	}

	// Use the first NAT gateway for the default route
	// All traffic will go through this gateway
	// Note: Hetzner Cloud networks support only one default route
	return c.addNATGatewayRoute(network, natGateways[0])
}

// addNATGatewayRoute adds a default route to the network via the NAT gateway
func (c *CreatorEnhanced) addNATGatewayRoute(network *hcloud.Network, natGateway *hcloud.Server) error {
	// Get the NAT gateway's private IP
	if len(natGateway.PrivateNet) == 0 {
		return fmt.Errorf("NAT gateway %s has no private IP address", natGateway.Name)
	}
	// Safe to access index 0 because we checked length above
	gatewayIP := natGateway.PrivateNet[0].IP

	// Refresh network data to get the latest routes
	// This is necessary because the network object passed to this function may have been
	// created earlier in the process and doesn't reflect routes added in previous runs
	refreshedNetwork, err := c.HetznerClient.GetNetwork(c.ctx, network.Name)
	if err != nil {
		return fmt.Errorf("failed to refresh network data: %w", err)
	}
	if refreshedNetwork == nil {
		return fmt.Errorf("network %s not found", network.Name)
	}

	// Check if route already exists
	for _, route := range refreshedNetwork.Routes {
		// Compare network destinations using CIDR string representation
		// This is the most reliable way to compare IPNet values
		if route.Destination != nil && route.Destination.String() == defaultRouteDestination.String() {
			if route.Gateway.Equal(gatewayIP) {
				util.LogInfo("Default route via NAT gateway already exists, skipping", "nat gateway")
				return nil
			}
			// Route exists but with different gateway - log a warning but skip to be idempotent
			util.LogWarning(fmt.Sprintf("Default route exists with different gateway (%s), skipping route addition", route.Gateway.String()), "nat gateway")
			return nil
		}
	}

	// Add route to network using pre-parsed destination
	return c.HetznerClient.AddRouteToNetwork(c.ctx, network, hcloud.NetworkAddRouteOpts{
		Route: hcloud.NetworkRoute{
			Destination: defaultRouteDestination,
			Gateway:     gatewayIP,
		},
	})
}

// createFirewall creates and applies firewall to the cluster
func (c *CreatorEnhanced) createFirewall(network *hcloud.Network, masters []*hcloud.Server) error {
	// Use the NetworkResourceManager to create firewall
	networkMgr := NewNetworkResourceManager(c.Config, c.HetznerClient)

	// Check if firewall already exists
	fwName := fmt.Sprintf("%s-firewall", c.Config.ClusterName)
	existingFw, err := c.HetznerClient.GetFirewall(c.ctx, fwName)
	if err == nil && existingFw != nil {
		util.LogInfo("Firewall already exists, using existing firewall", "firewall")
		return nil
	}

	// Create firewall with rules
	_, err = networkMgr.CreateClusterFirewall(network)
	if err != nil {
		return err
	}

	// Apply firewall to all cluster servers using label selector
	// The firewall is already applied via label selector in CreateClusterFirewall
	util.LogSuccess(fmt.Sprintf("Firewall created and will be applied to all nodes with cluster=%s label", c.Config.ClusterName), "firewall")

	return nil
}

// generateNATGatewayCloudInit generates cloud-init user data for NAT gateway
func (c *CreatorEnhanced) generateNATGatewayCloudInit() (string, error) {
	return cloudinit.GenerateNATGatewayCloudInit(c.Config.Networking.PrivateNetwork.Subnet)
}

// generateCloudInit generates cloud-init user data for servers
// If pool is provided, pool-specific settings are appended to root-level settings.
// Note: This uses additive behavior (global first, then pool additions).
// Global cluster parameters always come first, then per-pool parameters are appended.
func (c *CreatorEnhanced) generateCloudInit(pool *config.NodePool) (string, error) {
	// Merge packages: global first, then pool additions
	packages := append([]string{}, c.Config.AdditionalPackages...)
	if pool != nil && len(pool.AdditionalPackages) > 0 {
		packages = append(packages, pool.AdditionalPackages...)
	}

	// Merge pre-k3s commands: global first, then pool additions
	preK3sCommands := append([]string{}, c.Config.AdditionalPreK3sCommands...)
	if pool != nil && len(pool.AdditionalPreK3sCommands) > 0 {
		preK3sCommands = append(preK3sCommands, pool.AdditionalPreK3sCommands...)
	}

	// Merge post-k3s commands: global first, then pool additions
	postK3sCommands := append([]string{}, c.Config.AdditionalPostK3sCommands...)
	if pool != nil && len(pool.AdditionalPostK3sCommands) > 0 {
		postK3sCommands = append(postK3sCommands, pool.AdditionalPostK3sCommands...)
	}

	generator := cloudinit.NewGenerator(&cloudinit.Config{
		SSHPort:                   c.Config.Networking.SSH.Port,
		Packages:                  packages,
		AdditionalPreK3sCommands:  preK3sCommands,
		AdditionalPostK3sCommands: postK3sCommands,
		ClusterCIDR:               c.Config.Networking.ClusterCIDR,
		ServiceCIDR:               c.Config.Networking.ServiceCIDR,
		AllowedNetworksSSH:        c.Config.Networking.AllowedNetworks.SSH,
		AllowedNetworksAPI:        c.Config.Networking.AllowedNetworks.API,
	})

	cloudInitYAML, err := generator.Generate()
	if err != nil {
		return "", fmt.Errorf("failed to generate cloud-init: %w", err)
	}

	return cloudInitYAML, nil
}

// getOrCreatePlacementGroup gets an existing placement group or creates a new one
func (c *CreatorEnhanced) getOrCreatePlacementGroup(pgConfig *config.PlacementGroupConfig) (*hcloud.PlacementGroup, error) {
	if pgConfig == nil {
		return nil, nil
	}

	// Build placement group name with cluster prefix
	pgName := fmt.Sprintf("%s-%s", c.Config.ClusterName, pgConfig.Name)

	// Try to get existing placement group
	existingPG, err := c.HetznerClient.GetPlacementGroup(c.ctx, pgName)
	if err == nil && existingPG != nil {
		util.LogInfo(fmt.Sprintf("Using existing placement group: %s", pgName), "placement group")
		return existingPG, nil
	}

	// Create labels map for placement group
	labels := map[string]string{
		"cluster": c.Config.ClusterName,
		"managed": "kuberaptor",
	}
	for _, label := range pgConfig.Labels {
		labels[label.Key] = label.Value
	}

	// Create new placement group
	opts := hcloud.PlacementGroupCreateOpts{
		Name:   pgName,
		Type:   hcloud.PlacementGroupType(pgConfig.Type),
		Labels: labels,
	}

	util.LogInfo(fmt.Sprintf("Creating placement group: %s (type: %s)", pgName, pgConfig.Type), "placement group")
	pg, err := c.HetznerClient.CreatePlacementGroup(c.ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create placement group: %w", err)
	}

	util.LogSuccess(fmt.Sprintf("Placement group created: %s", pgName), "placement group")
	return pg, nil
}

// createMasterNodes creates master nodes in parallel
func (c *CreatorEnhanced) createMasterNodes(sshKey *hcloud.SSHKey, network *hcloud.Network) ([]*hcloud.Server, error) {
	// Generate cloud-init data once for all masters, using masters pool configuration
	cloudInitData, err := c.generateCloudInit(&c.Config.MastersPool.NodePool)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cloud-init: %w", err)
	}

	// Get or create placement group if configured
	var placementGroup *hcloud.PlacementGroup
	if c.Config.MastersPool.PlacementGroup != nil {
		placementGroup, err = c.getOrCreatePlacementGroup(c.Config.MastersPool.PlacementGroup)
		if err != nil {
			return nil, fmt.Errorf("failed to get or create placement group: %w", err)
		}
	}

	// Create a slice to hold masters and a mutex to protect it
	masters := make([]*hcloud.Server, c.Config.MastersPool.InstanceCount)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	// Create masters in parallel
	for i := 0; i < c.Config.MastersPool.InstanceCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			location := c.Config.MastersPool.Locations[index%len(c.Config.MastersPool.Locations)]
			nodeName := fmt.Sprintf("%s-master-%s-%d", c.Config.ClusterName, location, index+1)

			// Check if server already exists
			existingServer, err := c.HetznerClient.GetServer(c.ctx, nodeName)
			if err == nil && existingServer != nil {
				util.LogInfo(fmt.Sprintf("Master node already exists, using existing node: %s", nodeName), "master")
				mu.Lock()
				masters[index] = existingServer
				mu.Unlock()
				return
			}

			// Get server type
			serverType, err := c.HetznerClient.GetServerType(c.ctx, c.Config.MastersPool.InstanceType)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("failed to get server type: %w", err))
				mu.Unlock()
				return
			}

			// Get location
			loc, err := c.HetznerClient.GetLocation(c.ctx, location)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("failed to get location: %w", err))
				mu.Unlock()
				return
			}

			// Get image
			image, err := c.HetznerClient.GetImage(c.ctx, c.Config.Image)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("failed to get image: %w", err))
				mu.Unlock()
				return
			}

			// Prepare server options
			opts := hcloud.ServerCreateOpts{
				Name:       nodeName,
				ServerType: serverType,
				Image:      image,
				Location:   loc,
				SSHKeys:    []*hcloud.SSHKey{sshKey},
				UserData:   cloudInitData,
				Labels:     c.buildMasterHetznerServerLabels(location),
			}

			// Add network if enabled
			if network != nil {
				opts.Networks = []*hcloud.Network{network}
			}

			// Add placement group if configured
			if placementGroup != nil {
				opts.PlacementGroup = placementGroup
			}

			// Disable public network if NAT gateway is enabled
			if c.Config.Networking.PrivateNetwork.NATGateway != nil && c.Config.Networking.PrivateNetwork.NATGateway.Enabled {
				opts.PublicNet = &hcloud.ServerCreatePublicNet{
					EnableIPv4: false,
					EnableIPv6: false,
				}
			}

			// Create server
			util.LogInfo(fmt.Sprintf("Creating master: %s in %s", nodeName, location), "master")
			server, err := c.HetznerClient.CreateServer(c.ctx, opts)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("failed to create server %s: %w", nodeName, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			masters[index] = server
			mu.Unlock()
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Check for errors
	if len(errors) > 0 {
		return nil, fmt.Errorf("errors creating master nodes: %v", errors)
	}

	return masters, nil
}

// createWorkerNodesFromPools creates worker nodes from a specific set of pools in parallel
// This should only be called with static (non-autoscaling) pools
// Returns both the servers and a mapping of each server to its pool configuration
func (c *CreatorEnhanced) createWorkerNodesFromPools(sshKey *hcloud.SSHKey, network *hcloud.Network, pools []config.WorkerNodePool) ([]*hcloud.Server, map[string]config.WorkerNodePool, error) {
	// Calculate total workers across all pools
	totalWorkers := 0
	for _, pool := range pools {
		totalWorkers += pool.InstanceCount
	}

	// Create slices to hold workers and their pool mappings
	workers := make([]*hcloud.Server, 0, totalWorkers)
	workerToPool := make(map[string]config.WorkerNodePool)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	// Create workers in parallel
	for poolIdx, pool := range pools {
		// Generate cloud-init data for this specific pool
		cloudInitData, err := c.generateCloudInit(&pool.NodePool)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate cloud-init for pool: %w", err)
		}

		// Get or create placement group for this pool if configured
		var placementGroup *hcloud.PlacementGroup
		if pool.PlacementGroup != nil {
			placementGroup, err = c.getOrCreatePlacementGroup(pool.PlacementGroup)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to get or create placement group for pool: %w", err)
			}
		}

		// Use instance_count for node creation (pools passed here should be static only)
		nodeCount := pool.InstanceCount

		for i := 0; i < nodeCount; i++ {
			wg.Add(1)
			go func(pIdx int, p config.WorkerNodePool, nodeIdx int, cloudInit string, pg *hcloud.PlacementGroup) {
				defer wg.Done()

				poolName := p.Name
				if poolName == nil {
					defaultName := fmt.Sprintf("pool-%d", pIdx+1)
					poolName = &defaultName
				}

				// Get location using round-robin distribution across available locations
				// This matches the pattern used for master nodes
				location := p.Locations[nodeIdx%len(p.Locations)]
				nodeName := fmt.Sprintf("%s-worker-%s-%s-%d", c.Config.ClusterName, *poolName, location, nodeIdx+1)

				// Check if server already exists
				existingServer, err := c.HetznerClient.GetServer(c.ctx, nodeName)
				if err == nil && existingServer != nil {
					util.LogInfo(fmt.Sprintf("Worker node already exists, using existing node: %s", nodeName), "worker")
					mu.Lock()
					workers = append(workers, existingServer)
					workerToPool[existingServer.Name] = p
					mu.Unlock()
					return
				}

				// Get server type
				serverType, err := c.HetznerClient.GetServerType(c.ctx, p.InstanceType)
				if err != nil {
					mu.Lock()
					errors = append(errors, fmt.Errorf("failed to get server type: %w", err))
					mu.Unlock()
					return
				}

				// Retrieve location object from Hetzner API
				loc, err := c.HetznerClient.GetLocation(c.ctx, location)
				if err != nil {
					mu.Lock()
					errors = append(errors, fmt.Errorf("failed to get location: %w", err))
					mu.Unlock()
					return
				}

				// Get image
				image, err := c.HetznerClient.GetImage(c.ctx, c.Config.Image)
				if err != nil {
					mu.Lock()
					errors = append(errors, fmt.Errorf("failed to get image: %w", err))
					mu.Unlock()
					return
				}

				// Prepare server options
				opts := hcloud.ServerCreateOpts{
					Name:       nodeName,
					ServerType: serverType,
					Image:      image,
					Location:   loc,
					SSHKeys:    []*hcloud.SSHKey{sshKey},
					UserData:   cloudInit,
					Labels:     c.buildHetznerServerLabels(&p, *poolName, location),
				}

				// Add network if enabled
				if network != nil {
					opts.Networks = []*hcloud.Network{network}
				}

				// Add placement group if configured
				if pg != nil {
					opts.PlacementGroup = pg
				}

				// Disable public network if NAT gateway is enabled
				if c.Config.Networking.PrivateNetwork.NATGateway != nil && c.Config.Networking.PrivateNetwork.NATGateway.Enabled {
					opts.PublicNet = &hcloud.ServerCreatePublicNet{
						EnableIPv4: false,
						EnableIPv6: false,
					}
				}

				// Create server
				util.LogInfo(fmt.Sprintf("Creating worker: %s in %s", nodeName, location), "worker")
				server, err := c.HetznerClient.CreateServer(c.ctx, opts)
				if err != nil {
					mu.Lock()
					errors = append(errors, fmt.Errorf("failed to create server %s: %w", nodeName, err))
					mu.Unlock()
					return
				}

				mu.Lock()
				workers = append(workers, server)
				workerToPool[server.Name] = p
				mu.Unlock()
			}(poolIdx, pool, i, cloudInitData, placementGroup)
		}
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Check for errors
	if len(errors) > 0 {
		return nil, nil, fmt.Errorf("errors creating worker nodes: %v", errors)
	}

	return workers, workerToPool, nil
}

// waitForNodes waits for nodes to be ready
func (c *CreatorEnhanced) waitForNodes(servers []*hcloud.Server) error {
	for i, server := range servers {
		// First, wait for the server to reach "running" status
		err := c.HetznerClient.WaitForServerStatus(c.ctx, server, hcloud.ServerStatusRunning, 5*60*time.Second)
		if err != nil {
			return fmt.Errorf("server %s failed to start: %w", server.Name, err)
		}

		// CRITICAL: Refresh server data from API to get updated network information
		// This is essential when private networking is enabled, as the PrivateNet
		// field may not be populated in the initial server object
		refreshedServer, err := c.HetznerClient.GetServer(c.ctx, server.Name)
		if err != nil {
			return fmt.Errorf("failed to refresh server %s: %w", server.Name, err)
		}
		if refreshedServer == nil {
			return fmt.Errorf("server %s not found after creation", server.Name)
		}

		// Update the server in the list with refreshed data
		servers[i] = refreshedServer
		server = refreshedServer

		// Determine IP to use for SSH connection (always use public IP for external SSH access)
		ip, err := GetServerSSHIP(server)
		if err != nil {
			return err
		}

		err = c.SSHClient.WaitForInstance(c.ctx, ip, c.Config.Networking.SSH.Port, "echo ready", "ready", c.Config.Networking.SSH.UseAgent, 30)
		if err != nil {
			return fmt.Errorf("node %s not ready: %w", server.Name, err)
		}

		err = c.SSHClient.WaitForCloudInit(c.ctx, ip, c.Config.Networking.SSH.Port, c.Config.Networking.SSH.UseAgent)
		if err != nil {
			return fmt.Errorf("cloud-init failed on %s: %w", server.Name, err)
		}
	}
	return nil
}

// generateK3sAddonFlags generates addon-related flags for k3s based on configuration
func (c *CreatorEnhanced) generateK3sAddonFlags() string {
	var flags []string

	// Always disable local-storage (local-path) since we use hcloud-csi as default
	if c.Config.Addons.LocalPathStorageClass == nil || !c.Config.Addons.LocalPathStorageClass.Enabled {
		flags = append(flags, "--disable local-storage")
	}

	// Disable traefik unless explicitly enabled
	if c.Config.Addons.Traefik == nil || !c.Config.Addons.Traefik.Enabled {
		flags = append(flags, "--disable traefik")
	}

	// Disable servicelb unless explicitly enabled
	if c.Config.Addons.ServiceLB == nil || !c.Config.Addons.ServiceLB.Enabled {
		flags = append(flags, "--disable servicelb")
	}

	// Disable metrics-server unless explicitly enabled (we'll install it separately if needed)
	if c.Config.Addons.MetricsServer == nil || !c.Config.Addons.MetricsServer.Enabled {
		flags = append(flags, "--disable metrics-server")
	}

	// Enable embedded registry mirror if configured
	if c.Config.Addons.EmbeddedRegistryMirror != nil && c.Config.Addons.EmbeddedRegistryMirror.Enabled {
		flags = append(flags, "--embedded-registry")
	}

	return strings.Join(flags, " ")
}

// isK3sInstalled checks if k3s is already installed and running on a server
func (c *CreatorEnhanced) isK3sInstalled(ip string) bool {
	// Check if k3s service exists and is active
	checkCmd := "systemctl is-active k3s 2>/dev/null"
	output, err := c.SSHClient.Run(c.ctx, ip, c.Config.Networking.SSH.Port, checkCmd, c.Config.Networking.SSH.UseAgent)
	if err == nil && strings.TrimSpace(output) == "active" {
		return true
	}

	// Also check k3s-agent service
	checkCmd = "systemctl is-active k3s-agent 2>/dev/null"
	output, err = c.SSHClient.Run(c.ctx, ip, c.Config.Networking.SSH.Port, checkCmd, c.Config.Networking.SSH.UseAgent)
	if err == nil && strings.TrimSpace(output) == "active" {
		return true
	}

	return false
}

// waitForK3sService waits for k3s service to become active after installation
func (c *CreatorEnhanced) waitForK3sService(ip string, serviceName string, timeout time.Duration) error {
	// Validate serviceName to prevent command injection
	var checkCmd string
	switch serviceName {
	case "k3s":
		checkCmd = "systemctl is-active k3s 2>/dev/null"
	case "k3s-agent":
		checkCmd = "systemctl is-active k3s-agent 2>/dev/null"
	default:
		return fmt.Errorf("invalid service name: %s (expected 'k3s' or 'k3s-agent')", serviceName)
	}

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check for context cancellation
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}

		output, err := c.SSHClient.Run(c.ctx, ip, c.Config.Networking.SSH.Port, checkCmd, c.Config.Networking.SSH.UseAgent)
		if err == nil && strings.TrimSpace(output) == "active" {
			return nil
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("timeout waiting for %s service to become active", serviceName)
}

// waitForKubeconfig waits for kubeconfig file to be created
func (c *CreatorEnhanced) waitForKubeconfig(ip string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check for context cancellation
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}

		output, err := c.SSHClient.Run(c.ctx, ip, c.Config.Networking.SSH.Port, k3sKubeconfigCheckCmd, c.Config.Networking.SSH.UseAgent)
		if err == nil && strings.TrimSpace(output) == "exists" {
			return nil
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("timeout waiting for kubeconfig file to be created")
}

// waitForInternetConnectivity waits for the node to have internet connectivity
// This is especially important when NAT gateway is used and routing needs to be configured
func (c *CreatorEnhanced) waitForInternetConnectivity(ip string, timeout time.Duration) error {
	// Generate the connectivity test command from template
	testCmd, err := cloudinit.GenerateInternetConnectivityTestCommand()
	if err != nil {
		return fmt.Errorf("failed to generate connectivity test command: %w", err)
	}

	deadline := time.Now().Add(timeout)
	var lastErr error
	var lastOutput string

	for time.Now().Before(deadline) {
		// Check for context cancellation
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}

		output, err := c.SSHClient.Run(c.ctx, ip, c.Config.Networking.SSH.Port, testCmd, c.Config.Networking.SSH.UseAgent)
		if err == nil && strings.TrimSpace(output) == "connected" {
			return nil
		}

		// Save for debugging
		lastErr = err
		lastOutput = strings.TrimSpace(output)

		time.Sleep(3 * time.Second)
	}

	// Log debugging information on timeout
	if lastOutput != "" || lastErr != nil {
		util.LogError(fmt.Sprintf("Connectivity test output: %s, error: %v", lastOutput, lastErr), "network")
	}

	return fmt.Errorf("timeout waiting for internet connectivity (check network configuration and routing)")
}

// isNATGatewayEnabled checks if NAT gateway is configured and enabled
func (c *CreatorEnhanced) isNATGatewayEnabled() bool {
	return c.Config.Networking.PrivateNetwork.Enabled &&
		c.Config.Networking.PrivateNetwork.NATGateway != nil &&
		c.Config.Networking.PrivateNetwork.NATGateway.Enabled
}

// checkNATConnectivityIfNeeded verifies internet connectivity when NAT gateway is enabled
// This ensures routing configured via additional_pre_k3s_commands is active before k3s installation
func (c *CreatorEnhanced) checkNATConnectivityIfNeeded(ip string, nodeType string) error {
	if c.isNATGatewayEnabled() {
		util.LogInfo("Verifying internet connectivity via NAT gateway", nodeType)
		if err := c.waitForInternetConnectivity(ip, 2*time.Minute); err != nil {
			return fmt.Errorf("failed to verify internet connectivity: %w", err)
		}
		util.LogInfo("Internet connectivity verified", nodeType)
	}
	return nil
}

// installK3sOnMaster installs k3s on a master node using cluster-init mode
// All masters use the same command with --cluster-init flag, allowing K3s to
// automatically form an etcd cluster. This enables parallel installation of all masters.
func (c *CreatorEnhanced) installK3sOnMaster(server *hcloud.Server, allMasters []*hcloud.Server, apiLoadBalancers []*hcloud.LoadBalancer) error {
	ip, err := GetServerSSHIP(server)
	if err != nil {
		return err
	}

	// Check if k3s is already installed
	if c.isK3sInstalled(ip) {
		// Clear the current line to prevent overlap with spinner
		fmt.Print("\r\033[K")
		util.LogInfo(fmt.Sprintf("K3s already installed on %s, skipping installation", server.Name), "master")
		return nil
	}

	// If NAT gateway is enabled, verify internet connectivity before attempting k3s installation
	if err := c.checkNATConnectivityIfNeeded(ip, "master"); err != nil {
		return err
	}

	// Generate TLS SANs for all masters and API load balancers
	tlsSans, err := GenerateTLSSans(c.Config, allMasters, server, apiLoadBalancers)
	if err != nil {
		return fmt.Errorf("failed to generate TLS SANs: %w", err)
	}

	// Generate addon flags (disable/enable flags)
	addonFlags := c.generateK3sAddonFlags()

	// Generate flannel backend flags
	flannelBackendFlags, err := generateFlannelBackendFlags(c.Config, c.Config.K3sVersion)
	if err != nil {
		return fmt.Errorf("failed to generate flannel backend flags: %w", err)
	}

	// Build base command with --cluster-init for all masters
	baseArgs := "--cluster-init"
	if addonFlags != "" {
		baseArgs += " " + addonFlags
	}
	if flannelBackendFlags != "" {
		baseArgs += " " + flannelBackendFlags
	}

	// Add TLS SANs
	if tlsSans != "" {
		baseArgs += " " + tlsSans
	}

	// Add flannel-iface if private network is enabled
	if shouldConfigureFlannelInterface(c.Config) {
		// Detect the private network interface and add flannel-iface flag
		networkIface, err := c.detectPrivateNetworkInterface(ip)
		if err != nil {
			return fmt.Errorf("failed to detect private network interface: %w", err)
		}
		if networkIface != "" {
			baseArgs += fmt.Sprintf(" --flannel-iface=%s", networkIface)
		}
	}

	// Add etcd arguments if embedded etcd is configured
	if c.Config.Datastore.Mode == "etcd" && c.Config.Datastore.EmbeddedEtcd != nil {
		etcdArgs := c.Config.Datastore.EmbeddedEtcd.GenerateEtcdArgs()
		if etcdArgs != "" {
			baseArgs += " " + etcdArgs
		}
	}

	// Generate install command using the unified template
	installCmd, err := cloudinit.GenerateK3sInstallMasterCommand(c.Config.K3sVersion, c.k3sToken, baseArgs)
	if err != nil {
		return fmt.Errorf("failed to generate k3s install command: %w", err)
	}

	_, err = c.SSHClient.Run(c.ctx, ip, c.Config.Networking.SSH.Port, installCmd, c.Config.Networking.SSH.UseAgent)
	if err != nil {
		return fmt.Errorf("k3s install failed: %w", err)
	}

	// Wait for k3s service to become active
	util.LogInfo(fmt.Sprintf("Waiting for k3s service to start on %s", server.Name), "master")
	if err := c.waitForK3sService(ip, "k3s", 2*time.Minute); err != nil {
		return fmt.Errorf("k3s service failed to start: %w", err)
	}

	// Wait for kubeconfig file to be created on each master
	// This ensures the k3s cluster is fully initialized on this node before proceeding
	// Each master generates its own kubeconfig as part of the k3s installation
	if err := c.waitForKubeconfig(ip, 1*time.Minute); err != nil {
		return fmt.Errorf("kubeconfig file not created: %w", err)
	}

	return nil
}

// getNodeTokenFromMaster retrieves the node-token from the first master server
// This token is used for worker nodes to join the cluster and is regenerated by K3s
// when nodes are removed, making it the authoritative token for joining
func (c *CreatorEnhanced) getNodeTokenFromMaster(firstMaster *hcloud.Server) (string, error) {
	masterIP, err := GetServerSSHIP(firstMaster)
	if err != nil {
		return "", fmt.Errorf("failed to get master SSH IP for %s: %w", firstMaster.Name, err)
	}

	// Retrieve the node-token from the master server using the defined constant path
	cmd := fmt.Sprintf("sudo cat %s", k3sNodeTokenPath)
	output, err := c.SSHClient.Run(c.ctx, masterIP, c.Config.Networking.SSH.Port, cmd, c.Config.Networking.SSH.UseAgent)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve node-token from master %s (%s): %w", firstMaster.Name, masterIP, err)
	}

	nodeToken := strings.TrimSpace(output)
	if nodeToken == "" {
		return "", fmt.Errorf("node-token file is empty or missing on master server %s (%s)", firstMaster.Name, masterIP)
	}

	// Basic validation: K3s node-tokens typically contain "::" as a separator
	// This helps catch cases where the file contains unexpected content
	if !strings.Contains(nodeToken, "::") {
		return "", fmt.Errorf("node-token format appears invalid on master server %s (%s), expected token with '::' separator", firstMaster.Name, masterIP)
	}

	return nodeToken, nil
}

// installK3sOnWorkerWithToken installs k3s on a worker node with pool-specific labels and taints
// using a pre-retrieved node-token for efficiency when installing multiple workers
func (c *CreatorEnhanced) installK3sOnWorkerWithToken(server *hcloud.Server, firstMaster *hcloud.Server, pool *config.WorkerNodePool, nodeToken string) error {
	ip, err := GetServerSSHIP(server)
	if err != nil {
		return err
	}

	// Check if k3s is already installed
	if c.isK3sInstalled(ip) {
		// Clear the current line to prevent overlap with spinner
		fmt.Print("\r\033[K")
		util.LogInfo(fmt.Sprintf("K3s already installed on %s, skipping installation", server.Name), "worker")
		return nil
	}

	// If NAT gateway is enabled, verify internet connectivity before attempting k3s installation
	if err := c.checkNATConnectivityIfNeeded(ip, "worker"); err != nil {
		return err
	}

	firstMasterIP, err := GetServerIP(firstMaster, c.Config)
	if err != nil {
		return err
	}

	// Build base args with flannel-iface if private network is enabled
	baseArgs := ""
	if shouldConfigureFlannelInterface(c.Config) {
		// Detect the private network interface and add flannel-iface flag
		networkIface, err := c.detectPrivateNetworkInterface(ip)
		if err != nil {
			return fmt.Errorf("failed to detect private network interface: %w", err)
		}
		if networkIface != "" {
			baseArgs = fmt.Sprintf(" --flannel-iface=%s", networkIface)
		}
	}

	// Build node labels and taints from pool configuration
	labelsAndTaints := c.buildLabelsAndTaintsForWorker(pool)
	if labelsAndTaints != "" {
		baseArgs += " " + labelsAndTaints
	}

	// Generate install command using template with the provided node-token
	k3sURL := fmt.Sprintf("https://%s:6443", firstMasterIP)
	installCmd, err := cloudinit.GenerateK3sInstallWorkerCommand(c.Config.K3sVersion, nodeToken, k3sURL, baseArgs)
	if err != nil {
		return fmt.Errorf("failed to generate k3s install command: %w", err)
	}

	_, err = c.SSHClient.Run(c.ctx, ip, c.Config.Networking.SSH.Port, installCmd, c.Config.Networking.SSH.UseAgent)
	if err != nil {
		return fmt.Errorf("k3s install failed: %w", err)
	}

	// Wait for k3s-agent service to become active
	util.LogInfo("Waiting for k3s-agent service to start", "worker")
	if err := c.waitForK3sService(ip, "k3s-agent", 2*time.Minute); err != nil {
		return fmt.Errorf("k3s-agent service failed to start: %w", err)
	}

	return nil
}

// buildLabelsAndTaintsForWorker builds the --node-label and --node-taint flags for k3s worker installation
// This follows the same pattern used by the cluster autoscaler to ensure consistency
func (c *CreatorEnhanced) buildLabelsAndTaintsForWorker(pool *config.WorkerNodePool) string {
	if pool == nil {
		return ""
	}

	var parts []string

	// Build node labels using Kubernetes labels
	kubernetesLabels := pool.KubernetesLabels()
	if len(kubernetesLabels) > 0 {
		labelStrs := make([]string, 0, len(kubernetesLabels))
		for _, label := range kubernetesLabels {
			// Escape special characters to prevent shell injection
			key := util.EscapeShellArg(label.Key)
			value := util.EscapeShellArg(label.Value)
			labelStrs = append(labelStrs, fmt.Sprintf("%s=%s", key, value))
		}
		parts = append(parts, fmt.Sprintf("--node-label=%s", strings.Join(labelStrs, ",")))
	}

	// Build node taints using Kubernetes taints
	kubernetesTaints := pool.KubernetesTaints()
	if len(kubernetesTaints) > 0 {
		taintStrs := make([]string, 0, len(kubernetesTaints))
		for _, taint := range kubernetesTaints {
			// Escape special characters to prevent shell injection
			key := util.EscapeShellArg(taint.Key)
			value := util.EscapeShellArg(taint.Value)
			effect := util.EscapeShellArg(taint.Effect)
			taintStrs = append(taintStrs, fmt.Sprintf("%s=%s:%s", key, value, effect))
		}
		parts = append(parts, fmt.Sprintf("--node-taint=%s", strings.Join(taintStrs, ",")))
	}

	return strings.Join(parts, " ")
}

// buildHetznerServerLabels builds the Hetzner Cloud server labels for a worker pool
// This merges the default labels (cluster, role, pool, location, managed) with custom labels from the pool configuration
func (c *CreatorEnhanced) buildMasterHetznerServerLabels(location string) map[string]string {
	// Start with default labels
	labels := map[string]string{
		"cluster":  c.Config.ClusterName,
		"role":     "master",
		"location": location,
		"managed":  "kuberaptor",
	}

	// Add custom Hetzner labels from master pool configuration
	customLabels := c.Config.MastersPool.HetznerLabels()
	for _, label := range customLabels {
		// Custom labels can override defaults except for "managed" which is always set
		if label.Key != "managed" {
			labels[label.Key] = label.Value
		}
	}

	return labels
}

func (c *CreatorEnhanced) buildHetznerServerLabels(pool *config.WorkerNodePool, poolName string, location string) map[string]string {
	// Start with default labels
	labels := map[string]string{
		"cluster":  c.Config.ClusterName,
		"role":     "worker",
		"pool":     poolName,
		"location": location,
		"managed":  "kuberaptor",
	}

	// Add custom Hetzner labels from pool configuration
	if pool != nil {
		customLabels := pool.HetznerLabels()
		for _, label := range customLabels {
			// Custom labels can override defaults except for "managed" which is always set
			if label.Key != "managed" {
				labels[label.Key] = label.Value
			}
		}
	}

	return labels
}

func (c *CreatorEnhanced) buildNATGatewayHetznerServerLabels(location string) map[string]string {
	// Start with default labels
	labels := map[string]string{
		"cluster":  c.Config.ClusterName,
		"role":     "nat-gateway",
		"location": location,
		"managed":  "kuberaptor",
	}

	// Add custom Hetzner labels from NAT gateway configuration
	if c.Config.Networking.PrivateNetwork.NATGateway != nil {
		customLabels := c.Config.Networking.PrivateNetwork.NATGateway.HetznerLabels()
		for _, label := range customLabels {
			// Custom labels can override defaults except for "managed" which is always set
			if label.Key != "managed" {
				labels[label.Key] = label.Value
			}
		}
	}

	return labels
}

// retrieveKubeconfig retrieves kubeconfig from the first master and configures the API server address
// The API server address selection logic:
// 1. If API load balancers exist: use first load balancer's public IP (for first region)
// 2. Else if server has public IP: use server's public IP
// 3. Else if NAT gateway is enabled: use server's private IP (requires network access or SSH tunnel)
// 4. Else: error - no accessible API endpoint
func (c *CreatorEnhanced) retrieveKubeconfig(server *hcloud.Server, apiLoadBalancers []*hcloud.LoadBalancer) error {
	ip, err := GetServerSSHIP(server)
	if err != nil {
		return err
	}

	// Wait for kubeconfig file to be created (in case it's not ready yet)
	if err := c.waitForKubeconfig(ip, 1*time.Minute); err != nil {
		return err
	}

	// Get kubeconfig from server (using predefined command constant)
	kubeconfigContent, err := c.SSHClient.Run(c.ctx, ip, c.Config.Networking.SSH.Port, k3sKubeconfigReadCmd, c.Config.Networking.SSH.UseAgent)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Expand kubeconfig path
	kubeconfigPath, err := config.ExpandPath(c.Config.KubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to expand kubeconfig path: %w", err)
	}

	// Replace localhost with actual server IP in kubeconfig
	// Determine which IP to use for external API access
	var serverIP string

	// Priority 1: API load balancer IP (if configured and created)
	// Use the first API load balancer (typically for the first master's location)
	if len(apiLoadBalancers) > 0 && apiLoadBalancers[0] != nil && apiLoadBalancers[0].PublicNet.IPv4.IP != nil {
		serverIP = apiLoadBalancers[0].PublicNet.IPv4.IP.String()
		util.LogInfo(fmt.Sprintf("Using API load balancer IP for kubeconfig: %s (location: %s)", serverIP, apiLoadBalancers[0].Location.Name), "kubeconfig")

		// For multi-location deployments, log all API LB IPs for reference
		if len(apiLoadBalancers) > 1 {
			util.LogInfo(fmt.Sprintf("Additional API load balancers available (%d total):", len(apiLoadBalancers)), "kubeconfig")
			for _, lb := range apiLoadBalancers {
				if lb != nil && lb.PublicNet.IPv4.IP != nil {
					util.LogInfo(fmt.Sprintf("  %s: %s (location: %s)", lb.Name, lb.PublicNet.IPv4.IP.String(), lb.Location.Name), "kubeconfig")
				}
			}
		}
	} else if server.PublicNet.IPv4.IP != nil {
		// Priority 2: Server has public IP - use it directly
		serverIP = server.PublicNet.IPv4.IP.String()
		util.LogInfo(fmt.Sprintf("Using master public IP for kubeconfig: %s", serverIP), "kubeconfig")
	} else if c.Config.Networking.PrivateNetwork.Enabled &&
		c.Config.Networking.PrivateNetwork.NATGateway != nil &&
		c.Config.Networking.PrivateNetwork.NATGateway.Enabled {
		// Priority 3: Server has no public IP and NAT gateway is enabled
		// Use master's private IP - requires network access (VPN/tunnel) or kubectl via SSH
		if len(server.PrivateNet) > 0 {
			serverIP = server.PrivateNet[0].IP.String()
			util.LogWarning(fmt.Sprintf("Using master private IP for kubeconfig: %s", serverIP), "kubeconfig")
			util.LogWarning("API server is only accessible via private network. Ensure you have network access or use SSH tunnel.", "kubeconfig")
			util.LogWarning("Consider enabling 'api_load_balancer.enabled: true' for external access.", "kubeconfig")
		} else {
			return fmt.Errorf("server %s has no private IP address", server.Name)
		}
	} else {
		// Priority 4: No accessible endpoint
		return fmt.Errorf("server %s has no accessible IP address for API access (no public IP, no API load balancer, and NAT gateway is not enabled)", server.Name)
	}

	kubeconfigContent = strings.Replace(kubeconfigContent, "https://127.0.0.1:6443", fmt.Sprintf("https://%s:6443", serverIP), 1)

	// Replace "default" names with cluster name in kubeconfig
	kubeconfigContent = replaceKubeconfigNames(kubeconfigContent, c.Config.ClusterName)

	// Write kubeconfig to file
	if err := util.WriteToFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		return err
	}

	return nil
}

// replaceKubeconfigNames replaces "default" names in kubeconfig with the actual cluster name
// This includes cluster name, context name, user name, and references in contexts
//
// Pattern-based replacement is safe here because:
// 1. K3s-generated kubeconfig has a predictable structure with no comments
// 2. Certificate data is base64 encoded and won't contain these exact YAML patterns
// 3. The patterns are field-specific (e.g., "name: default", "cluster: default")
// 4. This avoids the overhead and complexity of parsing and re-serializing YAML
func replaceKubeconfigNames(kubeconfigContent, clusterName string) string {
	// Replace all occurrences of 'name: default' with 'name: <clusterName>'
	// This covers cluster name, context name, and user name
	kubeconfigContent = strings.ReplaceAll(kubeconfigContent, "name: default", fmt.Sprintf("name: %s", clusterName))

	// Replace context references: 'cluster: default' and 'user: default'
	kubeconfigContent = strings.ReplaceAll(kubeconfigContent, "cluster: default", fmt.Sprintf("cluster: %s", clusterName))
	kubeconfigContent = strings.ReplaceAll(kubeconfigContent, "user: default", fmt.Sprintf("user: %s", clusterName))

	// Replace current-context
	kubeconfigContent = strings.ReplaceAll(kubeconfigContent, "current-context: default", fmt.Sprintf("current-context: %s", clusterName))

	return kubeconfigContent
}

// installAddons installs cluster addons
func (c *CreatorEnhanced) installAddons(firstMaster *hcloud.Server, masters []*hcloud.Server, autoscalingPools []config.WorkerNodePool) error {
	// Get SSH IP (for external connections)
	masterSSHIP, err := GetServerSSHIP(firstMaster)
	if err != nil {
		return fmt.Errorf("failed to get master SSH IP: %w", err)
	}

	// Get cluster IP (for internal cluster communication)
	masterClusterIP, err := GetServerIP(firstMaster, c.Config)
	if err != nil {
		return fmt.Errorf("failed to get master cluster IP: %w", err)
	}

	installer := addons.NewInstaller(c.Config, c.SSHClient)
	return installer.InstallAll(firstMaster, masters, autoscalingPools, masterSSHIP, masterClusterIP, c.k3sToken)
}

// detectPrivateNetworkInterface detects the private network interface on a server
//
// The detection logic:
// - Looks for interfaces with MTU 1450 or 1280 (typical for Hetzner private networks)
// - Excludes virtual interfaces created by Cilium, Docker, Flannel, bridge, and veth
// - Returns the first matching interface name
//
// Expected network setup:
// - Hetzner Cloud private network attached to the server
// - Private network interface configured with MTU 1450 or 1280
// - Interface name typically follows pattern like 'ens10', 'eth1', etc.
func (c *CreatorEnhanced) detectPrivateNetworkInterface(ip string) (string, error) {
	// Command to detect private network interface
	// This matches the logic from templates/master_install_script.sh
	detectCmd := `ip -o link show | awk -F': ' '/mtu (1450|1280)/ {print $2}' | grep -Ev 'cilium|br|flannel|docker|veth' | head -n1`

	output, err := c.SSHClient.Run(c.ctx, ip, c.Config.Networking.SSH.Port, detectCmd, c.Config.Networking.SSH.UseAgent)
	if err != nil {
		return "", fmt.Errorf("failed to detect network interface: %w", err)
	}

	iface := strings.TrimSpace(output)
	if iface == "" {
		util.LogWarning("Could not detect private network interface, flannel will use default interface", "network")
		return "", nil
	}

	return iface, nil
}

// separateWorkerPools separates worker pools into static and autoscaling pools
func separateWorkerPools(pools []config.WorkerNodePool) (static []config.WorkerNodePool, autoscaling []config.WorkerNodePool) {
	for _, pool := range pools {
		if pool.AutoscalingEnabled() {
			autoscaling = append(autoscaling, pool)
		} else {
			static = append(static, pool)
		}
	}
	return static, autoscaling
}
