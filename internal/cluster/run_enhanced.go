package cluster

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/internal/util"
	"github.com/magenx/kuberaptor/pkg/hetzner"
)

// RunnerEnhanced handles running commands on cluster nodes with parallel execution
type RunnerEnhanced struct {
	Config        *config.Main
	HetznerClient *hetzner.Client
	SSHClient     *util.SSH
	ctx           context.Context
}

// NewRunnerEnhanced creates a new enhanced command runner
func NewRunnerEnhanced(cfg *config.Main, hetznerClient *hetzner.Client) (*RunnerEnhanced, error) {
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

	runner := &RunnerEnhanced{
		Config:        cfg,
		HetznerClient: hetznerClient,
		SSHClient:     sshClient,
		ctx:           context.Background(),
	}

	// Configure NAT gateway as bastion host if enabled
	if err := configureNATGatewayBastion(runner.ctx, runner.Config, runner.HetznerClient, runner.SSHClient, "run"); err != nil {
		return nil, fmt.Errorf("failed to configure NAT gateway bastion: %w", err)
	}

	return runner, nil
}

// RunCommand runs a command on all nodes or a specific instance with parallel execution
func (r *RunnerEnhanced) RunCommand(command string, instanceName string) error {
	if instanceName != "" {
		return r.runCommandOnInstance(command, instanceName)
	}
	return r.runCommandOnAllNodesParallel(command)
}

// requestUserConfirmation prompts the user to confirm an action
func (r *RunnerEnhanced) requestUserConfirmation(prompt string) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Type 'continue' to %s: ", prompt)

	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input != "continue" {
		return fmt.Errorf("operation cancelled")
	}

	fmt.Println()
	return nil
}

// printExecutionSummary prints a summary of nodes that will be affected
func (r *RunnerEnhanced) printExecutionSummary(servers []*hcloud.Server, actionDescription string) {
	fmt.Printf("Found %d instances in the cluster\n", len(servers))
	fmt.Println(actionDescription)
	fmt.Println()

	fmt.Println("Nodes that will be affected:")
	for _, server := range servers {
		ip, err := GetServerSSHIP(server)
		if err != nil || ip == "" {
			fmt.Printf("  - %s (no IP address - will be skipped)\n", server.Name)
		} else {
			fmt.Printf("  - %s (%s)\n", server.Name, ip)
		}
	}
	fmt.Println()
}

// printSingleInstanceExecutionSummary prints a summary for a single instance
func (r *RunnerEnhanced) printSingleInstanceExecutionSummary(server *hcloud.Server, actionDescription string) {
	fmt.Println("Found instance in the cluster")
	fmt.Println(actionDescription)
	fmt.Println()

	fmt.Println("Node that will be affected:")
	ip, err := GetServerSSHIP(server)
	if err != nil || ip == "" {
		fmt.Printf("  - %s (no IP address - will be skipped)\n", server.Name)
	} else {
		fmt.Printf("  - %s (%s)\n", server.Name, ip)
	}
	fmt.Println()
}

// RunScript runs a script on all nodes or a specific instance
func (r *RunnerEnhanced) RunScript(scriptPath string, instanceName string) error {
	// Validate script file
	if err := r.validateScriptFile(scriptPath); err != nil {
		return err
	}

	// Read script content
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read script: %w", err)
	}

	scriptName := filepath.Base(scriptPath)

	if instanceName != "" {
		return r.runScriptOnInstance(string(scriptContent), scriptName, instanceName)
	}
	return r.runScriptOnAllNodesParallel(string(scriptContent), scriptName)
}

// validateScriptFile validates that the script file exists and is readable
func (r *RunnerEnhanced) validateScriptFile(scriptPath string) error {
	info, err := os.Stat(scriptPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("script file '%s' does not exist", scriptPath)
		}
		return fmt.Errorf("failed to stat script file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("'%s' is not a file", scriptPath)
	}

	// Try to open file to check readability
	file, err := os.Open(scriptPath)
	if err != nil {
		return fmt.Errorf("script file '%s' is not readable: %w", scriptPath, err)
	}
	file.Close()

	return nil
}

// runCommandOnAllNodesParallel runs a command on all nodes in parallel
func (r *RunnerEnhanced) runCommandOnAllNodesParallel(command string) error {
	// Find all servers with cluster label
	clusterLabel := fmt.Sprintf("cluster=%s", r.Config.ClusterName)
	servers, err := r.HetznerClient.ListServers(r.ctx, hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: clusterLabel,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list servers: %w", err)
	}

	// Also find servers from autoscaling-enabled worker node pools
	autoscaledServers, err := findAutoscaledPoolServers(r.ctx, r.Config, r.HetznerClient)
	if err != nil {
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

	if len(allServers) == 0 {
		return fmt.Errorf("no servers found for cluster: %s", r.Config.ClusterName)
	}

	// Print execution summary
	r.printExecutionSummary(allServers, fmt.Sprintf("Command to execute: %s", command))

	// Request user confirmation
	if err := r.requestUserConfirmation("execute this command on all nodes"); err != nil {
		util.LogWarning("Command execution cancelled.", "run")
		return err
	}

	util.LogInfo("Running command on all nodes in parallel", "run")

	// Run command on each server in parallel
	type result struct {
		server *hcloud.Server
		output string
		err    error
	}

	results := make(chan result, len(allServers))
	var wg sync.WaitGroup

	for _, server := range allServers {
		wg.Add(1)
		go func(srv *hcloud.Server) {
			defer wg.Done()
			output, err := r.executeCommandOnServer(srv, command)
			results <- result{server: srv, output: output, err: err}
		}(server)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and display results
	var hasErrors bool
	var successCount, failCount int

	for res := range results {
		fmt.Printf("\n=== Instance: %s ===\n", res.server.Name)
		if res.err != nil {
			fmt.Printf("Command failed: %v\n\n", res.err)
			hasErrors = true
			failCount++
		} else {
			fmt.Println(res.output)
			fmt.Println("Command completed successfully")
			successCount++
		}
	}

	// Summary
	if hasErrors {
		util.LogWarning(fmt.Sprintf("Command execution completed: %d succeeded, %d failed", successCount, failCount), "run")
		return fmt.Errorf("command failed on %d node(s)", failCount)
	}

	util.LogSuccess(fmt.Sprintf("Command execution completed: %d succeeded", successCount), "run")
	return nil
}

// runCommandOnInstance runs a command on a specific instance
func (r *RunnerEnhanced) runCommandOnInstance(command string, instanceName string) error {
	server, err := r.HetznerClient.GetServer(r.ctx, instanceName)
	if err != nil || server == nil {
		return fmt.Errorf("instance not found: %s", instanceName)
	}

	// Print execution summary
	r.printSingleInstanceExecutionSummary(server, fmt.Sprintf("Command to execute: %s", command))

	// Request user confirmation
	if err := r.requestUserConfirmation("execute this command on instance"); err != nil {
		util.LogWarning("Command execution cancelled.", "run")
		return err
	}

	util.LogInfo(fmt.Sprintf("Running command on instance: %s", instanceName), "run")

	output, err := r.executeCommandOnServer(server, command)

	// Print formatted output
	fmt.Printf("\n=== Instance: %s ===\n", server.Name)
	if err != nil {
		fmt.Printf("Command failed: %v\n\n", err)
		return err
	}

	fmt.Println(output)
	fmt.Println("Command completed successfully")

	return nil
}

// executeCommandOnServer executes a command on a server and returns output
func (r *RunnerEnhanced) executeCommandOnServer(server *hcloud.Server, command string) (string, error) {
	// Get server IP for SSH connection
	ip, err := GetServerSSHIP(server)
	if err != nil {
		return "", err
	}

	// Execute command via SSH
	output, err := r.SSHClient.Run(r.ctx, ip, r.Config.Networking.SSH.Port, command, r.Config.Networking.SSH.UseAgent)
	if err != nil {
		return "", fmt.Errorf("command failed: %w", err)
	}

	return output, nil
}

// runScriptOnAllNodesParallel runs a script on all nodes in parallel
func (r *RunnerEnhanced) runScriptOnAllNodesParallel(scriptContent, scriptName string) error {
	// Find all servers with cluster label
	clusterLabel := fmt.Sprintf("cluster=%s", r.Config.ClusterName)
	servers, err := r.HetznerClient.ListServers(r.ctx, hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: clusterLabel,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list servers: %w", err)
	}

	// Also find servers from autoscaling-enabled worker node pools
	autoscaledServers, err := findAutoscaledPoolServers(r.ctx, r.Config, r.HetznerClient)
	if err != nil {
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

	if len(allServers) == 0 {
		return fmt.Errorf("no servers found for cluster: %s", r.Config.ClusterName)
	}

	// Print execution summary
	r.printExecutionSummary(allServers, fmt.Sprintf("Script to upload and execute: %s", scriptName))

	// Request user confirmation
	if err := r.requestUserConfirmation("upload and execute this script on all nodes"); err != nil {
		util.LogWarning("Script execution cancelled.", "run")
		return err
	}

	util.LogInfo("Running script on all nodes in parallel", "run")

	// Run script on each server in parallel
	type result struct {
		server *hcloud.Server
		output string
		err    error
	}

	results := make(chan result, len(allServers))
	var wg sync.WaitGroup

	for _, server := range allServers {
		wg.Add(1)
		go func(srv *hcloud.Server) {
			defer wg.Done()
			output, err := r.executeScriptOnServer(srv, scriptContent, scriptName)
			results <- result{server: srv, output: output, err: err}
		}(server)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and display results
	var hasErrors bool
	var successCount, failCount int

	for res := range results {
		fmt.Printf("\n=== Instance: %s ===\n", res.server.Name)
		if res.err != nil {
			fmt.Printf("Script failed: %v\n\n", res.err)
			hasErrors = true
			failCount++
		} else {
			if res.output != "" {
				fmt.Println(res.output)
			}
			fmt.Println("Script execution completed successfully")
			successCount++
		}
	}

	// Summary
	if hasErrors {
		util.LogWarning(fmt.Sprintf("Script execution completed: %d succeeded, %d failed", successCount, failCount), "run")
		return fmt.Errorf("script failed on %d node(s)", failCount)
	}

	util.LogSuccess(fmt.Sprintf("Script execution completed: %d succeeded", successCount), "run")
	return nil
}

// runScriptOnInstance runs a script on a specific instance
func (r *RunnerEnhanced) runScriptOnInstance(scriptContent, scriptName, instanceName string) error {
	server, err := r.HetznerClient.GetServer(r.ctx, instanceName)
	if err != nil || server == nil {
		return fmt.Errorf("instance not found: %s", instanceName)
	}

	// Print execution summary
	r.printSingleInstanceExecutionSummary(server, fmt.Sprintf("Script to upload and execute: %s", scriptName))

	// Request user confirmation
	if err := r.requestUserConfirmation("upload and execute this script on instance"); err != nil {
		util.LogWarning("Script execution cancelled.", "run")
		return err
	}

	util.LogInfo(fmt.Sprintf("Running script on instance: %s", instanceName), "run")

	output, err := r.executeScriptOnServer(server, scriptContent, scriptName)

	// Print formatted output
	fmt.Printf("\n=== Instance: %s ===\n", server.Name)
	if err != nil {
		fmt.Printf("Script failed: %v\n\n", err)
		return err
	}

	if output != "" {
		fmt.Println(output)
	}
	fmt.Println("Script execution completed successfully")

	return nil
}

// executeScriptOnServer uploads a script to a server, executes it, and cleans up
func (r *RunnerEnhanced) executeScriptOnServer(server *hcloud.Server, scriptContent, scriptName string) (string, error) {
	// Get server IP for SSH connection
	ip, err := GetServerSSHIP(server)
	if err != nil {
		return "", err
	}

	remoteScriptPath := fmt.Sprintf("/tmp/%s", scriptName)
	var allOutput []string

	// Upload script using heredoc with unique delimiter to prevent EOF conflicts
	uploadCommand := fmt.Sprintf("cat > %s << 'HEKSTER_SCRIPT_EOF'\n%s\nHEKSTER_SCRIPT_EOF", remoteScriptPath, scriptContent)
	uploadOutput, err := r.SSHClient.Run(r.ctx, ip, r.Config.Networking.SSH.Port, uploadCommand, r.Config.Networking.SSH.UseAgent)
	if err != nil {
		return "", fmt.Errorf("failed to upload script: %w", err)
	}
	if uploadOutput != "" {
		allOutput = append(allOutput, uploadOutput)
	}

	// Make script executable
	chmodCommand := fmt.Sprintf("chmod +x %s", remoteScriptPath)
	chmodOutput, err := r.SSHClient.Run(r.ctx, ip, r.Config.Networking.SSH.Port, chmodCommand, r.Config.Networking.SSH.UseAgent)
	if err != nil {
		r.cleanupScript(ip, remoteScriptPath)
		return "", fmt.Errorf("failed to make script executable: %w", err)
	}
	if chmodOutput != "" {
		allOutput = append(allOutput, chmodOutput)
	}

	// Execute script
	executeCommand := remoteScriptPath
	scriptOutput, err := r.SSHClient.Run(r.ctx, ip, r.Config.Networking.SSH.Port, executeCommand, r.Config.Networking.SSH.UseAgent)
	if err != nil {
		// Try to clean up even if execution failed
		r.cleanupScript(ip, remoteScriptPath)
		return "", fmt.Errorf("script execution failed: %w", err)
	}
	if scriptOutput != "" {
		allOutput = append(allOutput, scriptOutput)
	}

	// Clean up - remove the script
	r.cleanupScript(ip, remoteScriptPath)

	// Combine all output
	combinedOutput := strings.Join(allOutput, "\n")
	return combinedOutput, nil
}

// cleanupScript removes the temporary script file from the remote server
func (r *RunnerEnhanced) cleanupScript(ip, remoteScriptPath string) {
	// Ignore errors during cleanup - best effort only
	r.SSHClient.Run(r.ctx, ip, r.Config.Networking.SSH.Port, fmt.Sprintf("rm -f %s", remoteScriptPath), r.Config.Networking.SSH.UseAgent)
}
