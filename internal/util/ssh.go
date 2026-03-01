package util

import (
	"context"
	"crypto/md5"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

const (
	DefaultConnectTimeout = 15 * time.Second
	DefaultCommandTimeout = 30 * time.Second
	DefaultMaxAttempts    = 20
	DefaultRetryDelay     = 5 * time.Second
	CloudInitWaitTimeout  = 6 * time.Minute // Allows for 5 min wait in script + overhead
)

// cloudInitWaitScript contains the cloud-init wait script embedded from templates/cloud_init_wait_script.sh
//
//go:embed templates/cloud_init_wait_script.sh
var cloudInitWaitScript string

// SSH represents an SSH client wrapper
type SSH struct {
	privateKey  []byte // Private key content (loaded from path or inline)
	publicKey   []byte // Public key content (loaded from path or inline)
	bastionHost string // Bastion/jump host for ProxyJump
	bastionPort int    // Bastion SSH port
}

// NewSSH creates a new SSH client from file paths
func NewSSH(privateKeyPath, publicKeyPath string) (*SSH, error) {
	// Read keys from files
	privateKey, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key from %s: %w", privateKeyPath, err)
	}

	publicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key from %s: %w", publicKeyPath, err)
	}

	return &SSH{
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

// NewSSHFromKeys creates a new SSH client from key content
func NewSSHFromKeys(privateKey, publicKey []byte) *SSH {
	return &SSH{
		privateKey: privateKey,
		publicKey:  publicKey,
	}
}

// SetBastion configures a bastion/jump host for SSH connections
func (s *SSH) SetBastion(host string, port int) {
	s.bastionHost = host
	s.bastionPort = port
}

// GetPublicKey returns the public key content
func (s *SSH) GetPublicKey() ([]byte, error) {
	if len(s.publicKey) == 0 {
		return nil, fmt.Errorf("no public key available")
	}
	return s.publicKey, nil
}

// CalculateFingerprint calculates the MD5 fingerprint of a public key
// NOTE: MD5 is used here for compatibility with legacy SSH implementations
// and Hetzner Cloud API expectations. While MD5 is cryptographically weak,
// it's acceptable for this non-security-critical display/comparison purpose.
// Modern SSH implementations prefer SHA256 fingerprints.
func CalculateFingerprint(publicKeyData []byte) (string, error) {
	// Parse the public key
	parts := strings.Fields(string(publicKeyData))
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid public key format")
	}

	keyData, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode public key: %w", err)
	}

	// MD5 hash for legacy compatibility (not for security purposes)
	hash := md5.Sum(keyData)
	fingerprint := fmt.Sprintf("%x", hash)

	// Format as colon-separated pairs
	var result []string
	for i := 0; i < len(fingerprint); i += 2 {
		result = append(result, fingerprint[i:i+2])
	}

	return strings.Join(result, ":"), nil
}

// CalculateFingerprintFromPath calculates the MD5 fingerprint of a public key from a file path
func CalculateFingerprintFromPath(publicKeyPath string) (string, error) {
	data, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read public key: %w", err)
	}
	return CalculateFingerprint(data)
}

// getSSHConfig returns SSH client configuration
func (s *SSH) getSSHConfig(useAgent bool) (*ssh.ClientConfig, error) {
	var authMethods []ssh.AuthMethod

	if useAgent {
		// Try to use SSH agent
		if agentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
			authMethods = append(authMethods, ssh.PublicKeysCallback(agent.NewClient(agentConn).Signers))
		}
	}

	// Parse private key from memory
	if len(s.privateKey) == 0 {
		return nil, fmt.Errorf("no private key available")
	}

	signer, err := ssh.ParsePrivateKey(s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	authMethods = append(authMethods, ssh.PublicKeys(signer))

	// Setup host key verification using known_hosts
	hostKeyCallback, err := s.getHostKeyCallback()
	if err != nil {
		return nil, fmt.Errorf("failed to setup host key verification: %w", err)
	}

	config := &ssh.ClientConfig{
		User:            "root",
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         DefaultConnectTimeout,
	}

	return config, nil
}

// getHostKeyCallback returns a host key callback that uses known_hosts file
// for verification. If the known_hosts file doesn't exist, it creates one
// and accepts new host keys on first connection.
func (s *SSH) getHostKeyCallback() (ssh.HostKeyCallback, error) {
	// Determine the known_hosts file path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	knownHostsPath := filepath.Join(homeDir, ".ssh", "known_hosts")

	// Ensure .ssh directory exists
	sshDir := filepath.Dir(knownHostsPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	// Create known_hosts file if it doesn't exist
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		file, err := os.OpenFile(knownHostsPath, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to create known_hosts file: %w", err)
		}
		file.Close()
	}

	// Create a callback using knownhosts package
	hostKeyCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create known_hosts callback: %w", err)
	}

	// Wrap the callback to handle new hosts and key changes gracefully
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// Try the standard known_hosts verification first
		err := hostKeyCallback(hostname, remote, key)
		if err == nil {
			// Host key is already known and matches
			return nil
		}

		// Check if this is a new host (not in known_hosts)
		var keyErr *knownhosts.KeyError
		if !errors.As(err, &keyErr) {
			// Some other error occurred
			return err
		}

		// If host is not known, add it to known_hosts
		if len(keyErr.Want) == 0 {
			// New host - add the key to known_hosts
			f, fileErr := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_WRONLY, 0600)
			if fileErr != nil {
				return fmt.Errorf("failed to open known_hosts for writing: %w", fileErr)
			}
			defer f.Close()

			// Format: hostname algorithm base64-key
			line := knownhosts.Line([]string{hostname}, key)
			if _, fileErr = f.WriteString(line + "\n"); fileErr != nil {
				return fmt.Errorf("failed to write to known_hosts: %w", fileErr)
			}

			return nil
		}

		// Host key has changed - this could be a security issue
		// For ephemeral infrastructure, we log a warning but accept the new key
		fmt.Fprintf(os.Stderr, "WARNING: Host key for %s has changed. This is expected for ephemeral infrastructure.\n", hostname)
		fmt.Fprintf(os.Stderr, "Previous key fingerprint(s): %v\n", keyErr.Want)
		fmt.Fprintf(os.Stderr, "New key fingerprint: %s\n", ssh.FingerprintSHA256(key))

		// Update the known_hosts file with the new key
		// Read all lines
		content, readErr := os.ReadFile(knownHostsPath)
		if readErr != nil {
			return fmt.Errorf("failed to read known_hosts: %w", readErr)
		}

		// Remove old entries for this host
		lines := strings.Split(string(content), "\n")
		var newLines []string
		for _, line := range lines {
			if line == "" {
				continue
			}
			// Parse the hostname(s) from the known_hosts line
			// Format is: hostname[,hostname...] keytype base64key [comment]
			parts := strings.Fields(line)
			if len(parts) < 3 {
				// Malformed line, keep it
				newLines = append(newLines, line)
				continue
			}
			// Check if our hostname is in the host list
			hosts := strings.Split(parts[0], ",")
			foundMatch := false
			for _, h := range hosts {
				// Strip port specification if present: [hostname]:port or hostname:port
				cleanHost := strings.TrimPrefix(h, "[")
				if idx := strings.Index(cleanHost, "]:"); idx != -1 {
					cleanHost = cleanHost[:idx]
				} else if idx := strings.LastIndex(cleanHost, ":"); idx != -1 {
					// Only strip :port if it's not an IPv6 address
					if !strings.Contains(cleanHost, ":") || strings.Count(cleanHost, ":") > 1 {
						// IPv6 address, keep as is
					} else {
						cleanHost = cleanHost[:idx]
					}
				}
				if cleanHost == hostname {
					foundMatch = true
					break
				}
			}
			// Keep lines that don't match this hostname
			if !foundMatch {
				newLines = append(newLines, line)
			}
		}

		// Add the new key
		newLines = append(newLines, knownhosts.Line([]string{hostname}, key))

		// Write back to file
		var builder strings.Builder
		builder.WriteString(strings.Join(newLines, "\n"))
		builder.WriteString("\n")
		if writeErr := os.WriteFile(knownHostsPath, []byte(builder.String()), 0600); writeErr != nil {
			return fmt.Errorf("failed to update known_hosts: %w", writeErr)
		}

		return nil
	}, nil
}

// getClient returns an SSH client, optionally through a bastion host
func (s *SSH) getClient(config *ssh.ClientConfig, host string, port int) (*ssh.Client, error) {
	addr := fmt.Sprintf("%s:%d", host, port)

	// If bastion host is configured, connect through it
	if s.bastionHost != "" {
		// Connect to bastion
		bastionAddr := fmt.Sprintf("%s:%d", s.bastionHost, s.bastionPort)
		bastionClient, err := ssh.Dial("tcp", bastionAddr, config)
		if err != nil {
			return nil, fmt.Errorf("failed to dial bastion %s: %w", bastionAddr, err)
		}

		// Connect to target through bastion
		conn, err := bastionClient.Dial("tcp", addr)
		if err != nil {
			bastionClient.Close()
			return nil, fmt.Errorf("failed to dial %s through bastion: %w", addr, err)
		}

		// Create SSH connection over the tunneled connection
		ncc, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
		if err != nil {
			conn.Close()
			bastionClient.Close()
			return nil, fmt.Errorf("failed to create client connection through bastion: %w", err)
		}

		// Note: The bastion client connection is embedded in the returned SSH client
		// and will be cleaned up when the client is closed
		return ssh.NewClient(ncc, chans, reqs), nil
	}

	// Direct connection (no bastion)
	return ssh.Dial("tcp", addr, config)
}

// Run executes a command on a remote host via SSH
func (s *SSH) Run(ctx context.Context, host string, port int, command string, useAgent bool) (string, error) {
	config, err := s.getSSHConfig(useAgent)
	if err != nil {
		return "", err
	}

	// Connect to the remote host (possibly through bastion)
	client, err := s.getClient(config, host, port)
	if err != nil {
		return "", err
	}
	defer client.Close()

	// Create a session
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Execute the command with timeout
	type result struct {
		output string
		err    error
	}

	resultChan := make(chan result, 1)

	go func() {
		output, err := session.CombinedOutput(command)
		resultChan <- result{output: string(output), err: err}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-resultChan:
		return res.output, res.err
	}
}

// RunWithOutput executes a command and streams output
func (s *SSH) RunWithOutput(ctx context.Context, host string, port int, command string, useAgent bool, prefix string) error {
	config, err := s.getSSHConfig(useAgent)
	if err != nil {
		return err
	}

	// Connect to the remote host (possibly through bastion)
	client, err := s.getClient(config, host, port)
	if err != nil {
		return err
	}
	defer client.Close()

	// Create a session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Setup pipes
	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := session.Start(command); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Stream output
	done := make(chan error, 1)
	go func() {
		done <- session.Wait()
	}()

	// Read output
	go StreamWithPrefix(stdout, prefix, false)
	go StreamWithPrefix(stderr, prefix, true)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// WaitForInstance waits for an instance to be ready by running a test command
func (s *SSH) WaitForInstance(ctx context.Context, host string, port int, testCommand string, expectedResult string, useAgent bool, maxAttempts int) error {
	if maxAttempts == 0 {
		maxAttempts = DefaultMaxAttempts
	}

	for i := 0; i < maxAttempts; i++ {
		// Create context with timeout for this attempt
		attemptCtx, cancel := context.WithTimeout(ctx, DefaultCommandTimeout)

		result, err := s.Run(attemptCtx, host, port, testCommand, useAgent)
		cancel()

		if err == nil {
			result = strings.TrimSpace(result)
			if result == expectedResult {
				return nil
			}
		}

		if i < maxAttempts-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(DefaultRetryDelay):
				// Continue to next attempt
			}
		}
	}

	return fmt.Errorf("instance not ready after %d attempts", maxAttempts)
}

// WaitForCloudInit waits for cloud-init to complete on a remote host
func (s *SSH) WaitForCloudInit(ctx context.Context, host string, port int, useAgent bool) error {
	// Execute the wait script with a timeout that allows for the script's 5 min wait + overhead
	waitCtx, cancel := context.WithTimeout(ctx, CloudInitWaitTimeout)
	defer cancel()

	_, err := s.Run(waitCtx, host, port, cloudInitWaitScript, useAgent)
	if err != nil {
		return fmt.Errorf("cloud-init did not complete: %w", err)
	}

	return nil
}

// CopyFile copies a file to the remote host via SCP
func (s *SSH) CopyFile(ctx context.Context, host string, port int, localPath string, remotePath string, useAgent bool) error {
	config, err := s.getSSHConfig(useAgent)
	if err != nil {
		return err
	}

	// Connect to the remote host (possibly through bastion)
	client, err := s.getClient(config, host, port)
	if err != nil {
		return err
	}
	defer client.Close()

	// Read local file
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	// Create remote file
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Write file content via SSH
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	if err := session.Start(fmt.Sprintf("cat > %s", remotePath)); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	if _, err := stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	stdin.Close()

	return session.Wait()
}
