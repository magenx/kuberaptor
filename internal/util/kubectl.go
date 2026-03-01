package util

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/magenx/kuberaptor/internal/config"
)

// KubectlClient wraps kubectl operations using local kubectl with kubeconfig
type KubectlClient struct {
	kubeconfigPath string
	ctx            context.Context
}

// NewKubectlClient creates a new kubectl client
func NewKubectlClient(kubeconfigPath string) *KubectlClient {
	// Expand the path once during creation
	expandedPath, err := config.ExpandPath(kubeconfigPath)
	if err != nil {
		// If expansion fails, use the original path
		expandedPath = kubeconfigPath
	}

	return &KubectlClient{
		kubeconfigPath: expandedPath,
		ctx:            context.Background(),
	}
}

// Apply applies a manifest from URL
func (k *KubectlClient) Apply(manifestURL string) error {
	cmd := exec.CommandContext(k.ctx, "kubectl", "apply", "--validate=false", "-f", manifestURL)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", k.kubeconfigPath))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// ApplyManifest applies a manifest from stdin
func (k *KubectlClient) ApplyManifest(manifest string) error {
	ctx, cancel := context.WithTimeout(k.ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", "apply", "--validate=false", "-f", "-")
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", k.kubeconfigPath))

	// Create a pipe to feed the manifest
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Capture output
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start kubectl: %w", err)
	}

	// Write the manifest to stdin
	if _, err := stdin.Write([]byte(manifest)); err != nil {
		stdin.Close()
		return fmt.Errorf("failed to write manifest: %w", err)
	}
	stdin.Close()

	// Wait for command to complete (with timeout from context)
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("kubectl apply failed: %w\nOutput: %s", err, output.String())
	}

	return nil
}

// Get executes a kubectl get command and returns the output
func (k *KubectlClient) Get(args ...string) (string, error) {
	cmdArgs := append([]string{"get"}, args...)
	cmd := exec.CommandContext(k.ctx, "kubectl", cmdArgs...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", k.kubeconfigPath))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("kubectl get failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// ClusterInfo executes kubectl cluster-info
func (k *KubectlClient) ClusterInfo() (string, error) {
	cmd := exec.CommandContext(k.ctx, "kubectl", "cluster-info")
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", k.kubeconfigPath))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("kubectl cluster-info failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// ResourceExists checks if a resource exists in the cluster
// resourceType: deployment, daemonset, namespace, etc.
// name: the resource name
// namespace: the namespace (use "" for cluster-scoped resources)
func (k *KubectlClient) ResourceExists(resourceType, name, namespace string) bool {
	ctx, cancel := context.WithTimeout(k.ctx, 10*time.Second)
	defer cancel()

	args := []string{"get", resourceType, name}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	args = append(args, "--ignore-not-found")

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", k.kubeconfigPath))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	// Trim whitespace and check if output is empty
	// When --ignore-not-found is used and the resource doesn't exist, output is empty
	trimmedOutput := strings.TrimSpace(string(output))
	return len(trimmedOutput) > 0
}
