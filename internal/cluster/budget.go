// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package cluster

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/internal/util"
	"github.com/magenx/kuberaptor/pkg/hetzner"
)

const (
	// hoursPerMonth is the average number of hours in a month (24 * 30.42)
	hoursPerMonth = 730.0
)

// BudgetCalculator handles budget calculation
type BudgetCalculator struct {
	Config        *config.Main
	HetznerClient *hetzner.Client
	ctx           context.Context
}

// ResourceCost represents the cost of a resource
type ResourceCost struct {
	Type         string
	Name         string
	ResourceType string
	HourlyPrice  float64
	MonthlyPrice float64
	Currency     string
}

// NewBudgetCalculator creates a new budget calculator
func NewBudgetCalculator(cfg *config.Main, hetznerClient *hetzner.Client) *BudgetCalculator {
	return &BudgetCalculator{
		Config:        cfg,
		HetznerClient: hetznerClient,
		ctx:           context.Background(),
	}
}

// Run executes the budget calculation
func (b *BudgetCalculator) Run() error {
	util.LogInfo("Calculating cluster budget", b.Config.ClusterName)

	clusterLabel := fmt.Sprintf("cluster=%s", b.Config.ClusterName)

	var costs []ResourceCost
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	// Fetch servers and their costs
	wg.Add(1)
	go func() {
		defer wg.Done()
		serverCosts, err := b.getServerCosts(clusterLabel)
		if err != nil {
			errChan <- fmt.Errorf("failed to get server costs: %w", err)
			return
		}
		mu.Lock()
		costs = append(costs, serverCosts...)
		mu.Unlock()
	}()

	// Fetch load balancers and their costs
	wg.Add(1)
	go func() {
		defer wg.Done()
		lbCosts, err := b.getLoadBalancerCosts(clusterLabel)
		if err != nil {
			errChan <- fmt.Errorf("failed to get load balancer costs: %w", err)
			return
		}
		mu.Lock()
		costs = append(costs, lbCosts...)
		mu.Unlock()
	}()

	// Fetch network costs (network traffic is free)
	// Networks themselves don't have a cost, but we list them for completeness
	wg.Add(1)
	go func() {
		defer wg.Done()
		networkCosts, err := b.getNetworkCosts(clusterLabel)
		if err != nil {
			errChan <- fmt.Errorf("failed to get network costs: %w", err)
			return
		}
		mu.Lock()
		costs = append(costs, networkCosts...)
		mu.Unlock()
	}()

	// Fetch firewall costs (firewalls are free)
	wg.Add(1)
	go func() {
		defer wg.Done()
		firewallCosts, err := b.getFirewallCosts(clusterLabel)
		if err != nil {
			errChan <- fmt.Errorf("failed to get firewall costs: %w", err)
			return
		}
		mu.Lock()
		costs = append(costs, firewallCosts...)
		mu.Unlock()
	}()

	// Fetch SSH key costs (SSH keys are free)
	wg.Add(1)
	go func() {
		defer wg.Done()
		sshKeyCosts, err := b.getSSHKeyCosts(clusterLabel)
		if err != nil {
			errChan <- fmt.Errorf("failed to get SSH key costs: %w", err)
			return
		}
		mu.Lock()
		costs = append(costs, sshKeyCosts...)
		mu.Unlock()
	}()

	// Fetch volume costs
	wg.Add(1)
	go func() {
		defer wg.Done()
		volumeCosts, err := b.getVolumeCosts(clusterLabel)
		if err != nil {
			errChan <- fmt.Errorf("failed to get volume costs: %w", err)
			return
		}
		mu.Lock()
		costs = append(costs, volumeCosts...)
		mu.Unlock()
	}()

	// Fetch floating IP costs
	wg.Add(1)
	go func() {
		defer wg.Done()
		floatingIPCosts, err := b.getFloatingIPCosts(clusterLabel)
		if err != nil {
			errChan <- fmt.Errorf("failed to get floating IP costs: %w", err)
			return
		}
		mu.Lock()
		costs = append(costs, floatingIPCosts...)
		mu.Unlock()
	}()

	// Fetch primary IP costs
	wg.Add(1)
	go func() {
		defer wg.Done()
		primaryIPCosts, err := b.getPrimaryIPCosts(clusterLabel)
		if err != nil {
			errChan <- fmt.Errorf("failed to get primary IP costs: %w", err)
			return
		}
		mu.Lock()
		costs = append(costs, primaryIPCosts...)
		mu.Unlock()
	}()

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	// Display the budget
	b.displayBudget(costs)

	return nil
}

// getServerCosts retrieves server costs
func (b *BudgetCalculator) getServerCosts(clusterLabel string) ([]ResourceCost, error) {
	spinner := util.NewSpinner("Finding servers", "servers")
	spinner.Start()

	servers, err := b.HetznerClient.ListServers(b.ctx, hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: clusterLabel,
		},
	})
	if err != nil {
		spinner.Stop(true)
		return nil, err
	}

	// Also find servers from autoscaling-enabled worker node pools
	// These are created by the cluster autoscaler and have the HCloudNodeGroupLabel
	autoscaledServers, err := b.findAutoscaledPoolServers()
	if err != nil {
		spinner.Stop(true)
		return nil, fmt.Errorf("failed to find autoscaled pool servers: %w", err)
	}

	// Combine all servers
	allServers := append(servers, autoscaledServers...)

	spinner.Stop(true)

	var costs []ResourceCost
	for _, server := range allServers {
		// Get server type pricing for the server's location
		serverType := server.ServerType
		var pricing *hcloud.ServerTypeLocationPricing
		for _, p := range serverType.Pricings {
			if p.Location.Name == server.Datacenter.Location.Name {
				pricing = &p
				break
			}
		}

		if pricing != nil {
			hourly, err := strconv.ParseFloat(pricing.Hourly.Gross, 64)
			if err != nil {
				util.LogWarning(fmt.Sprintf("Failed to parse hourly price for server %s: %v", server.Name, err), "")
				hourly = 0
			}
			monthly, err := strconv.ParseFloat(pricing.Monthly.Gross, 64)
			if err != nil {
				util.LogWarning(fmt.Sprintf("Failed to parse monthly price for server %s: %v", server.Name, err), "")
				monthly = 0
			}

			costs = append(costs, ResourceCost{
				Type:         "Server",
				Name:         server.Name,
				ResourceType: serverType.Name,
				HourlyPrice:  hourly,
				MonthlyPrice: monthly,
				Currency:     pricing.Monthly.Currency,
			})
		}
	}

	return costs, nil
}

// findAutoscaledPoolServers finds servers created by the cluster autoscaler
// These servers have the HCloudNodeGroupLabel label instead of the cluster label
func (b *BudgetCalculator) findAutoscaledPoolServers() ([]*hcloud.Server, error) {
	var allServers []*hcloud.Server

	// Iterate through all worker node pools
	for _, pool := range b.Config.WorkerNodePools {
		// Only process autoscaling-enabled pools
		if !pool.AutoscalingEnabled() {
			continue
		}

		// Build the node pool name (must match the name used by cluster autoscaler)
		poolName := pool.BuildNodePoolName(b.Config.ClusterName)

		// For multi-location pools, search each location-specific node group
		for _, location := range pool.Locations {
			searchPoolName := poolName
			// For multi-location pools, append location suffix
			if len(pool.Locations) > 1 {
				searchPoolName = fmt.Sprintf("%s-%s", poolName, location)
			}

			// Search for servers with the HCloudNodeGroupLabel
			nodeGroupLabel := fmt.Sprintf("%s=%s", HCloudNodeGroupLabel, searchPoolName)
			servers, err := b.HetznerClient.ListServers(b.ctx, hcloud.ServerListOpts{
				ListOpts: hcloud.ListOpts{
					LabelSelector: nodeGroupLabel,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to list servers for node group %s: %w", searchPoolName, err)
			}

			allServers = append(allServers, servers...)
		}
	}

	return allServers, nil
}

// getLoadBalancerCosts retrieves load balancer costs
func (b *BudgetCalculator) getLoadBalancerCosts(clusterLabel string) ([]ResourceCost, error) {
	spinner := util.NewSpinner("Finding load balancers", "load balancers")
	spinner.Start()

	hcloudClient := b.HetznerClient.GetHCloudClient()
	loadBalancers, err := hcloudClient.LoadBalancer.AllWithOpts(b.ctx, hcloud.LoadBalancerListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: clusterLabel,
		},
	})
	if err != nil {
		spinner.Stop(true)
		return nil, err
	}

	spinner.Stop(true)

	var costs []ResourceCost
	for _, lb := range loadBalancers {
		// Get load balancer type pricing for the load balancer's location
		lbType := lb.LoadBalancerType
		var pricing *hcloud.LoadBalancerTypeLocationPricing
		for _, p := range lbType.Pricings {
			if p.Location.Name == lb.Location.Name {
				pricing = &p
				break
			}
		}

		if pricing != nil {
			hourly, err := strconv.ParseFloat(pricing.Hourly.Gross, 64)
			if err != nil {
				util.LogWarning(fmt.Sprintf("Failed to parse hourly price for load balancer %s: %v", lb.Name, err), "")
				hourly = 0
			}
			monthly, err := strconv.ParseFloat(pricing.Monthly.Gross, 64)
			if err != nil {
				util.LogWarning(fmt.Sprintf("Failed to parse monthly price for load balancer %s: %v", lb.Name, err), "")
				monthly = 0
			}

			costs = append(costs, ResourceCost{
				Type:         "Load Balancer",
				Name:         lb.Name,
				ResourceType: lbType.Name,
				HourlyPrice:  hourly,
				MonthlyPrice: monthly,
				Currency:     pricing.Monthly.Currency,
			})
		}
	}

	return costs, nil
}

// getNetworkCosts retrieves network costs (networks are free)
func (b *BudgetCalculator) getNetworkCosts(clusterLabel string) ([]ResourceCost, error) {
	spinner := util.NewSpinner("Finding networks", "networks")
	spinner.Start()

	hcloudClient := b.HetznerClient.GetHCloudClient()
	networks, err := hcloudClient.Network.AllWithOpts(b.ctx, hcloud.NetworkListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: clusterLabel,
		},
	})
	if err != nil {
		spinner.Stop(true)
		return nil, err
	}

	spinner.Stop(true)

	var costs []ResourceCost
	for _, network := range networks {
		costs = append(costs, ResourceCost{
			Type:         "Network",
			Name:         network.Name,
			ResourceType: network.IPRange.String(),
			HourlyPrice:  0,
			MonthlyPrice: 0,
			Currency:     "EUR",
		})
	}

	return costs, nil
}

// getFirewallCosts retrieves firewall costs (firewalls are free)
func (b *BudgetCalculator) getFirewallCosts(clusterLabel string) ([]ResourceCost, error) {
	spinner := util.NewSpinner("Finding firewalls", "firewalls")
	spinner.Start()

	hcloudClient := b.HetznerClient.GetHCloudClient()
	firewalls, err := hcloudClient.Firewall.AllWithOpts(b.ctx, hcloud.FirewallListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: clusterLabel,
		},
	})
	if err != nil {
		spinner.Stop(true)
		return nil, err
	}

	spinner.Stop(true)

	var costs []ResourceCost
	for _, firewall := range firewalls {
		costs = append(costs, ResourceCost{
			Type:         "Firewall",
			Name:         firewall.Name,
			ResourceType: fmt.Sprintf("%d rules", len(firewall.Rules)),
			HourlyPrice:  0,
			MonthlyPrice: 0,
			Currency:     "EUR",
		})
	}

	return costs, nil
}

// getSSHKeyCosts retrieves SSH key costs (SSH keys are free)
func (b *BudgetCalculator) getSSHKeyCosts(clusterLabel string) ([]ResourceCost, error) {
	spinner := util.NewSpinner("Finding SSH keys", "SSH keys")
	spinner.Start()

	hcloudClient := b.HetznerClient.GetHCloudClient()
	sshKeys, err := hcloudClient.SSHKey.AllWithOpts(b.ctx, hcloud.SSHKeyListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: clusterLabel,
		},
	})
	if err != nil {
		spinner.Stop(true)
		return nil, err
	}

	spinner.Stop(true)

	var costs []ResourceCost
	for _, sshKey := range sshKeys {
		fingerprint := sshKey.Fingerprint
		if len(fingerprint) > 16 {
			fingerprint = fingerprint[:16] + "..."
		}
		costs = append(costs, ResourceCost{
			Type:         "SSH Key",
			Name:         sshKey.Name,
			ResourceType: fingerprint,
			HourlyPrice:  0,
			MonthlyPrice: 0,
			Currency:     "EUR",
		})
	}

	return costs, nil
}

// getVolumeCosts retrieves volume costs
func (b *BudgetCalculator) getVolumeCosts(clusterLabel string) ([]ResourceCost, error) {
	spinner := util.NewSpinner("Finding volumes", "volumes")
	spinner.Start()

	hcloudClient := b.HetznerClient.GetHCloudClient()
	volumes, err := hcloudClient.Volume.AllWithOpts(b.ctx, hcloud.VolumeListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: clusterLabel,
		},
	})
	if err != nil {
		spinner.Stop(true)
		return nil, err
	}

	spinner.Stop(true)

	// Get pricing information
	pricing, _, err := hcloudClient.Pricing.Get(b.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing information: %w", err)
	}

	var costs []ResourceCost
	for _, volume := range volumes {
		// Calculate monthly cost based on volume size
		pricePerGB, err := strconv.ParseFloat(pricing.Volume.PerGBMonthly.Gross, 64)
		if err != nil {
			util.LogWarning(fmt.Sprintf("Failed to parse volume price per GB: %v", err), "")
			pricePerGB = 0
		}

		monthlyPrice := float64(volume.Size) * pricePerGB
		hourlyPrice := monthlyPrice / hoursPerMonth

		costs = append(costs, ResourceCost{
			Type:         "Volume",
			Name:         volume.Name,
			ResourceType: fmt.Sprintf("%dGB", volume.Size),
			HourlyPrice:  hourlyPrice,
			MonthlyPrice: monthlyPrice,
			Currency:     pricing.Currency,
		})
	}

	return costs, nil
}

// getFloatingIPCosts retrieves floating IP costs
func (b *BudgetCalculator) getFloatingIPCosts(clusterLabel string) ([]ResourceCost, error) {
	spinner := util.NewSpinner("Finding floating IPs", "floating IPs")
	spinner.Start()

	hcloudClient := b.HetznerClient.GetHCloudClient()
	floatingIPs, err := hcloudClient.FloatingIP.AllWithOpts(b.ctx, hcloud.FloatingIPListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: clusterLabel,
		},
	})
	if err != nil {
		spinner.Stop(true)
		return nil, err
	}

	spinner.Stop(true)

	// Get pricing information
	pricing, _, err := hcloudClient.Pricing.Get(b.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing information: %w", err)
	}

	var costs []ResourceCost
	for _, floatingIP := range floatingIPs {
		// Find pricing for this floating IP type and location
		var monthlyPrice float64
		var currency string = pricing.Currency

		for _, typePricing := range pricing.FloatingIPs {
			if typePricing.Type == floatingIP.Type {
				// Find pricing for the home location
				for _, locPricing := range typePricing.Pricings {
					if floatingIP.HomeLocation != nil && locPricing.Location.Name == floatingIP.HomeLocation.Name {
						price, err := strconv.ParseFloat(locPricing.Monthly.Gross, 64)
						if err != nil {
							util.LogWarning(fmt.Sprintf("Failed to parse floating IP price for %s: %v", floatingIP.Name, err), "")
							price = 0
						}
						monthlyPrice = price
						break
					}
				}
				break
			}
		}

		hourlyPrice := monthlyPrice / hoursPerMonth

		ipAddress := floatingIP.IP.String()
		costs = append(costs, ResourceCost{
			Type:         "Floating IP",
			Name:         floatingIP.Name,
			ResourceType: fmt.Sprintf("%s (%s)", floatingIP.Type, ipAddress),
			HourlyPrice:  hourlyPrice,
			MonthlyPrice: monthlyPrice,
			Currency:     currency,
		})
	}

	return costs, nil
}

// getPrimaryIPCosts retrieves primary IP costs
func (b *BudgetCalculator) getPrimaryIPCosts(clusterLabel string) ([]ResourceCost, error) {
	spinner := util.NewSpinner("Finding primary IPs", "primary IPs")
	spinner.Start()

	hcloudClient := b.HetznerClient.GetHCloudClient()
	primaryIPs, err := hcloudClient.PrimaryIP.AllWithOpts(b.ctx, hcloud.PrimaryIPListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: clusterLabel,
		},
	})
	if err != nil {
		spinner.Stop(true)
		return nil, err
	}

	spinner.Stop(true)

	// Get pricing information
	pricing, _, err := hcloudClient.Pricing.Get(b.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing information: %w", err)
	}

	var costs []ResourceCost
	for _, primaryIP := range primaryIPs {
		// Find pricing for this primary IP type and location
		var hourlyPrice, monthlyPrice float64
		var currency string = pricing.Currency

		for _, typePricing := range pricing.PrimaryIPs {
			if typePricing.Type == string(primaryIP.Type) {
				// Find pricing for the location
				for _, locPricing := range typePricing.Pricings {
					if primaryIP.Location != nil && locPricing.Location == primaryIP.Location.Name {
						hourly, err := strconv.ParseFloat(locPricing.Hourly.Gross, 64)
						if err != nil {
							util.LogWarning(fmt.Sprintf("Failed to parse primary IP hourly price for %s: %v", primaryIP.Name, err), "")
							hourly = 0
						}
						monthly, err := strconv.ParseFloat(locPricing.Monthly.Gross, 64)
						if err != nil {
							util.LogWarning(fmt.Sprintf("Failed to parse primary IP monthly price for %s: %v", primaryIP.Name, err), "")
							monthly = 0
						}
						hourlyPrice = hourly
						monthlyPrice = monthly
						break
					}
				}
				break
			}
		}

		ipAddress := primaryIP.IP.String()
		costs = append(costs, ResourceCost{
			Type:         "Primary IP",
			Name:         primaryIP.Name,
			ResourceType: fmt.Sprintf("%s (%s)", primaryIP.Type, ipAddress),
			HourlyPrice:  hourlyPrice,
			MonthlyPrice: monthlyPrice,
			Currency:     currency,
		})
	}

	return costs, nil
}

// displayBudget displays the budget information
func (b *BudgetCalculator) displayBudget(costs []ResourceCost) {
	// Group costs by type
	grouped := make(map[string][]ResourceCost)
	for _, cost := range costs {
		grouped[cost.Type] = append(grouped[cost.Type], cost)
	}

	// Display grouped costs
	fmt.Println("\x1b[1;34m  CLUSTER BUDGET ESTIMATE  \x1b[0m")
	fmt.Println()

	totalMonthly := 0.0
	currency := "EUR"

	// Define order of resource types
	resourceOrder := []string{"Server", "Load Balancer", "Volume", "Floating IP", "Primary IP", "Network", "Firewall", "SSH Key"}

	for _, resourceType := range resourceOrder {
		resources, exists := grouped[resourceType]
		if !exists || len(resources) == 0 {
			continue
		}

		fmt.Printf("\x1b[1;33m%s:\x1b[0m\n", resourceType+"s")
		for _, resource := range resources {
			if resource.Currency != "" {
				currency = resource.Currency
			}

			if resource.MonthlyPrice > 0 {
				fmt.Printf("  %-40s %-20s €%-8.2f €%-8.2f/month\n",
					resource.Name,
					resource.ResourceType,
					resource.HourlyPrice,
					resource.MonthlyPrice)
			} else {
				fmt.Printf("  %-40s %-20s %-9s %-9s\n",
					resource.Name,
					resource.ResourceType,
					"free",
					"free")
			}

			totalMonthly += resource.MonthlyPrice
		}
		fmt.Println()
	}

	// Display total
	fmt.Printf("\x1b[1;32mEstimated project total:\x1b[0m %-50s \n", fmt.Sprintf("€%.2f/month", totalMonthly))

	if totalMonthly > 0 {
		fmt.Printf("\n\x1b[90mNote: Prices are shown in %s including VAT and may vary slightly based on actual usage.\x1b[0m\n", currency)
		fmt.Printf("\x1b[90mAdditional charges may apply for network traffic above included limits.\x1b[0m\n")
		fmt.Println()
	}
}
