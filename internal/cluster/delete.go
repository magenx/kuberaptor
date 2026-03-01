package cluster

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/internal/util"
	"github.com/magenx/kuberaptor/pkg/hetzner"
)

// Deleter handles cluster deletion
type Deleter struct {
	Config        *config.Main
	HetznerClient *hetzner.Client
	Force         bool
	ctx           context.Context
}

// NewDeleter creates a new cluster deleter
func NewDeleter(cfg *config.Main, hetznerClient *hetzner.Client, force bool) *Deleter {
	return &Deleter{
		Config:        cfg,
		HetznerClient: hetznerClient,
		Force:         force,
		ctx:           context.Background(),
	}
}

// Run executes the cluster deletion process
func (d *Deleter) Run() error {
	util.LogInfo("Starting cluster deletion", d.Config.ClusterName)

	// Confirm deletion if not forced
	if !d.Force {
		// Request cluster name confirmation
		if err := d.requestClusterNameConfirmation(); err != nil {
			return err
		}

		// Check protection against deletion
		if d.Config.ProtectAgainstDeletion {
			util.LogError("Cluster cannot be deleted. If you are sure about this, disable the protection by setting `protect_against_deletion` to `false` in the config file. Aborting deletion.", "")
			return fmt.Errorf("cluster is protected against deletion")
		}
	}

	// Track errors during deletion
	var deletionErrors []string

	// Find all resources with cluster label
	clusterLabel := fmt.Sprintf("cluster=%s", d.Config.ClusterName)

	// Step 1: Delete servers first
	spinner := util.NewSpinner("Finding and deleting servers", "servers")
	spinner.Start()
	servers, err := d.HetznerClient.ListServers(d.ctx, hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: clusterLabel,
		},
	})
	if err != nil {
		spinner.Stop(true)
		return fmt.Errorf("failed to list servers: %w", err)
	}

	// Also find servers from autoscaling-enabled worker node pools
	// These are created by the cluster autoscaler and have the HCloudNodeGroupLabel
	autoscaledServers, err := findAutoscaledPoolServers(d.ctx, d.Config, d.HetznerClient)
	if err != nil {
		spinner.Stop(true)
		return fmt.Errorf("failed to find autoscaled pool servers: %w", err)
	}

	// Merge servers, avoiding duplicates
	serverMap := make(map[int64]*hcloud.Server)
	for _, server := range servers {
		serverMap[server.ID] = server
	}
	for _, server := range autoscaledServers {
		serverMap[server.ID] = server
	}

	// Convert map back to slice
	allServers := make([]*hcloud.Server, 0, len(serverMap))
	for _, server := range serverMap {
		allServers = append(allServers, server)
	}

	spinner.Stop(true)

	// Log found servers
	if len(allServers) == 0 {
		util.LogInfo("No servers found", "servers")
	} else {
		util.LogInfo(fmt.Sprintf("Found %d server(s):", len(allServers)), "servers")
		for _, server := range allServers {
			fmt.Printf("  - %s\n", server.Name)
		}

		// Delete servers in parallel for improved performance
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, server := range allServers {
			wg.Add(1)
			go func(srv *hcloud.Server) {
				defer wg.Done()

				util.LogInfo(fmt.Sprintf("Deleting server: %s", srv.Name), "servers")
				if err := d.HetznerClient.DeleteServer(d.ctx, srv); err != nil {
					errMsg := fmt.Sprintf("Failed to delete server %s: %v", srv.Name, err)
					util.LogError(errMsg, "servers")
					mu.Lock()
					deletionErrors = append(deletionErrors, errMsg)
					mu.Unlock()
				} else {
					util.LogSuccess(fmt.Sprintf("Deleted server: %s", srv.Name), "servers")
				}
			}(server)
		}

		// Wait for all server deletions to complete
		wg.Wait()
	}

	util.LogSuccess(fmt.Sprintf("Completed deletion of %d server(s)", len(allServers)), "servers")

	// Step 2: Delete load balancers
	// Find all load balancers using cluster labels (more reliable than hardcoded names)
	util.LogInfo("Finding and deleting load balancers", "load balancer")
	clusterLbLabel := fmt.Sprintf("cluster=%s,managed=kuberaptor", d.Config.ClusterName)
	loadBalancers, err := d.HetznerClient.ListLoadBalancers(d.ctx, hcloud.LoadBalancerListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: clusterLbLabel,
		},
	})
	if err != nil {
		errMsg := fmt.Sprintf("Failed to list load balancers: %v", err)
		util.LogError(errMsg, "load balancer")
		deletionErrors = append(deletionErrors, errMsg)
	} else {
		if len(loadBalancers) == 0 {
			util.LogInfo("No load balancers found", "load balancer")
		} else {
			util.LogInfo(fmt.Sprintf("Found %d load balancer(s):", len(loadBalancers)), "load balancer")
			for _, lb := range loadBalancers {
				role := "unknown"
				location := "unknown"
				if lb.Labels != nil {
					if r, ok := lb.Labels["role"]; ok {
						role = r
					}
					if l, ok := lb.Labels["location"]; ok {
						location = l
					}
				}
				fmt.Printf("  - %s (role: %s, location: %s)\n", lb.Name, role, location)
			}

			// Delete load balancers
			for _, lb := range loadBalancers {
				util.LogInfo(fmt.Sprintf("Deleting load balancer: %s", lb.Name), "load balancer")
				if err := d.HetznerClient.DeleteLoadBalancer(d.ctx, lb); err != nil {
					errMsg := fmt.Sprintf("Failed to delete load balancer %s: %v", lb.Name, err)
					util.LogError(errMsg, "load balancer")
					deletionErrors = append(deletionErrors, errMsg)
				} else {
					util.LogSuccess(fmt.Sprintf("Load balancer deleted: %s", lb.Name), "load balancer")
				}
			}
		}
	}

	// Step 3: Delete SSL certificate (if it was created)
	// Must be deleted AFTER load balancers that use it
	if d.Config.SSLCertificate.Enabled {
		certName := d.Config.Domain
		if d.Config.SSLCertificate.Name != "" {
			certName = d.Config.SSLCertificate.Name
		}

		// Retrieve certificate information
		cert, err := d.HetznerClient.GetCertificate(d.ctx, certName)
		isManagedByCluster := err == nil && cert != nil && cert.Labels["cluster"] == d.Config.ClusterName && cert.Labels["managed"] == "kuberaptor"

		// Check if SSL certificate should be preserved to avoid Let's Encrypt rate limits
		if d.Config.SSLCertificate.Preserve {
			// Verify that the certificate exists and is managed by this cluster
			if isManagedByCluster {
				util.LogInfo("SSL certificate preservation is enabled, skipping deletion to avoid Let's Encrypt rate limits", "ssl")
				util.LogInfo(fmt.Sprintf("Certificate '%s' will be reused when the cluster is recreated", certName), "ssl")
			} else if err == nil && cert != nil {
				util.LogInfo("SSL certificate preservation is enabled but certificate is not managed by this cluster", "ssl")
			} else {
				util.LogInfo("SSL certificate preservation is enabled but certificate does not exist (nothing to preserve)", "ssl")
			}
		} else {
			// Delete the certificate if it's managed by this cluster
			util.LogInfo("Finding and deleting SSL certificate", "ssl")
			if isManagedByCluster {
				if err := d.HetznerClient.DeleteCertificate(d.ctx, cert); err != nil {
					errMsg := fmt.Sprintf("Failed to delete SSL certificate: %v", err)
					util.LogError(errMsg, "ssl")
					deletionErrors = append(deletionErrors, errMsg)
				} else {
					util.LogSuccess("SSL certificate deleted", "ssl")
				}
			} else if err == nil && cert != nil {
				util.LogInfo("SSL certificate exists but is not managed by this cluster, skipping deletion", "ssl")
			}
		}
	}

	// Step 4: Delete DNS zone (if it was created)
	if d.Config.DNSZone.Enabled && d.Config.Domain != "" {
		util.LogInfo("Finding and deleting DNS zone", "dns")
		zoneName := d.Config.Domain
		if d.Config.DNSZone.Name != "" {
			zoneName = d.Config.DNSZone.Name
		}
		zone, err := d.HetznerClient.GetZone(d.ctx, zoneName)
		if err == nil && zone != nil {
			// Check if zone is managed by this cluster
			if zone.Labels["cluster"] == d.Config.ClusterName && zone.Labels["managed"] == "kuberaptor" {
				if err := d.HetznerClient.DeleteZone(d.ctx, zone); err != nil {
					errMsg := fmt.Sprintf("Failed to delete DNS zone: %v", err)
					util.LogError(errMsg, "dns")
					deletionErrors = append(deletionErrors, errMsg)
				} else {
					util.LogSuccess("DNS zone deleted", "dns")
				}
			} else {
				util.LogInfo("DNS zone exists but is not managed by this cluster, skipping deletion", "dns")
			}
		}
	}

	// Step 5: Delete network (after servers and load balancers that might be using it)
	if d.Config.Networking.PrivateNetwork.Enabled {
		util.LogInfo("Finding and deleting network", "network")
		// Use cluster name as network name (matching creation logic)
		networkName := d.Config.ClusterName
		network, err := d.HetznerClient.GetNetwork(d.ctx, networkName)
		if err == nil && network != nil {
			if err := d.HetznerClient.DeleteNetwork(d.ctx, network); err != nil {
				errMsg := fmt.Sprintf("Failed to delete network: %v", err)
				util.LogError(errMsg, "network")
				deletionErrors = append(deletionErrors, errMsg)
			} else {
				util.LogSuccess("Network deleted", "network")
			}
		}
	}

	// Step 6: Delete firewalls (after all other resources, before SSH key)
	util.LogInfo("Finding and deleting firewalls", "firewall")
	firewallName := fmt.Sprintf("%s-firewall", d.Config.ClusterName)
	firewall, err := d.HetznerClient.GetFirewall(d.ctx, firewallName)
	if err == nil && firewall != nil {
		if err := d.HetznerClient.DeleteFirewall(d.ctx, firewall); err != nil {
			errMsg := fmt.Sprintf("Failed to delete firewall: %v", err)
			util.LogError(errMsg, "firewall")
			deletionErrors = append(deletionErrors, errMsg)
		} else {
			util.LogSuccess("Firewall deleted", "firewall")
		}
	}

	// Step 7: Delete SSH key (last, no dependencies)
	util.LogInfo("Finding and deleting SSH key", "ssh key")
	keyName := fmt.Sprintf("%s-ssh-key", d.Config.ClusterName)
	sshKey, err := d.HetznerClient.GetSSHKey(d.ctx, keyName)
	if err == nil && sshKey != nil {
		if err := d.HetznerClient.DeleteSSHKey(d.ctx, sshKey); err != nil {
			errMsg := fmt.Sprintf("Failed to delete SSH key: %v", err)
			util.LogError(errMsg, "ssh key")
			deletionErrors = append(deletionErrors, errMsg)
		} else {
			util.LogSuccess("SSH key deleted", "ssh key")
		}
	}

	// Step 8: Delete placement groups (after all servers are deleted)
	util.LogInfo("Finding and deleting placement groups", "placement group")
	clusterPGLabel := fmt.Sprintf("cluster=%s,managed=kuberaptor", d.Config.ClusterName)
	placementGroups, err := d.HetznerClient.ListPlacementGroups(d.ctx, hcloud.PlacementGroupListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: clusterPGLabel,
		},
	})
	if err != nil {
		errMsg := fmt.Sprintf("Failed to list placement groups: %v", err)
		util.LogError(errMsg, "placement group")
		deletionErrors = append(deletionErrors, errMsg)
	} else {
		if len(placementGroups) == 0 {
			util.LogInfo("No placement groups found", "placement group")
		} else {
			util.LogInfo(fmt.Sprintf("Found %d placement group(s):", len(placementGroups)), "placement group")
			for _, pg := range placementGroups {
				fmt.Printf("  - %s (type: %s)\n", pg.Name, pg.Type)
			}

			// Delete placement groups
			for _, pg := range placementGroups {
				util.LogInfo(fmt.Sprintf("Deleting placement group: %s", pg.Name), "placement group")
				if err := d.HetznerClient.DeletePlacementGroup(d.ctx, pg); err != nil {
					errMsg := fmt.Sprintf("Failed to delete placement group %s: %v", pg.Name, err)
					util.LogError(errMsg, "placement group")
					deletionErrors = append(deletionErrors, errMsg)
				} else {
					util.LogSuccess(fmt.Sprintf("Placement group deleted: %s", pg.Name), "placement group")
				}
			}
		}
	}

	fmt.Println()

	// Report final status based on whether errors occurred
	if len(deletionErrors) > 0 {
		util.LogError("Cluster deletion completed with errors!", d.Config.ClusterName)
		util.LogWarning("The following resources failed to delete:", "")
		for _, errMsg := range deletionErrors {
			fmt.Printf("  - %s\n", errMsg)
		}
		return fmt.Errorf("cluster deletion completed with %d error(s)", len(deletionErrors))
	}

	// Clean up kubeconfig file if it exists
	d.cleanupKubeconfig()

	util.LogSuccess("Cluster deletion completed successfully!", d.Config.ClusterName)
	return nil
}

// requestClusterNameConfirmation prompts the user to confirm deletion by typing the cluster name
func (d *Deleter) requestClusterNameConfirmation() error {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Please enter the cluster name to confirm that you want to delete it: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)

		if input == "" {
			util.LogError("Input cannot be empty. Please enter the cluster name.", "")
			continue
		}

		if input != d.Config.ClusterName {
			util.LogError(fmt.Sprintf("Cluster name '%s' does not match expected '%s'. Aborting deletion.", input, d.Config.ClusterName), "")
			return fmt.Errorf("cluster name confirmation failed")
		}

		break
	}

	return nil
}

// cleanupKubeconfig removes the kubeconfig file if it exists
func (d *Deleter) cleanupKubeconfig() {
	kubeconfigPath, err := config.ExpandPath(d.Config.KubeconfigPath)
	if err != nil {
		util.LogWarning(fmt.Sprintf("Failed to expand kubeconfig path: %v", err), "kubeconfig")
		return
	}

	if _, err := os.Stat(kubeconfigPath); err == nil {
		if err := os.Remove(kubeconfigPath); err != nil {
			util.LogWarning(fmt.Sprintf("Failed to delete kubeconfig file: %v", err), "kubeconfig")
		} else {
			util.LogSuccess("Kubeconfig file deleted", "kubeconfig")
		}
	}
}
