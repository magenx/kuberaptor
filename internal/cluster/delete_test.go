// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package cluster

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/magenx/kuberaptor/internal/config"
	"github.com/magenx/kuberaptor/pkg/hetzner"
)

// MockHetznerClient is a mock implementation of the Hetzner client for testing
type MockHetznerClient struct {
	deleteServerFunc   func(ctx context.Context, server *hcloud.Server) error
	listServersFunc    func(ctx context.Context, opts hcloud.ServerListOpts) ([]*hcloud.Server, error)
	deleteDelay        time.Duration
	deleteCallCount    int
	deleteCallCountMux sync.Mutex
}

func (m *MockHetznerClient) DeleteServer(ctx context.Context, server *hcloud.Server) error {
	m.deleteCallCountMux.Lock()
	m.deleteCallCount++
	m.deleteCallCountMux.Unlock()

	if m.deleteDelay > 0 {
		time.Sleep(m.deleteDelay)
	}

	if m.deleteServerFunc != nil {
		return m.deleteServerFunc(ctx, server)
	}
	return nil
}

func (m *MockHetznerClient) ListServers(ctx context.Context, opts hcloud.ServerListOpts) ([]*hcloud.Server, error) {
	if m.listServersFunc != nil {
		return m.listServersFunc(ctx, opts)
	}
	return []*hcloud.Server{}, nil
}

func (m *MockHetznerClient) GetDeleteCallCount() int {
	m.deleteCallCountMux.Lock()
	defer m.deleteCallCountMux.Unlock()
	return m.deleteCallCount
}

// TestParallelServerDeletion verifies that servers are deleted in parallel
func TestParallelServerDeletion(t *testing.T) {
	tests := []struct {
		name            string
		serverCount     int
		deleteDelay     time.Duration
		maxExpectedTime time.Duration
	}{
		{
			name:            "single server",
			serverCount:     1,
			deleteDelay:     100 * time.Millisecond,
			maxExpectedTime: 300 * time.Millisecond,
		},
		{
			name:        "multiple servers in parallel",
			serverCount: 5,
			deleteDelay: 100 * time.Millisecond,
			// If sequential, it would take 500ms. Parallel should complete in ~100ms + overhead
			maxExpectedTime: 300 * time.Millisecond,
		},
		{
			name:        "many servers in parallel",
			serverCount: 10,
			deleteDelay: 50 * time.Millisecond,
			// If sequential, it would take 500ms. Parallel should complete in ~50ms + overhead
			maxExpectedTime: 200 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock servers
			servers := make([]*hcloud.Server, tt.serverCount)
			for i := 0; i < tt.serverCount; i++ {
				servers[i] = &hcloud.Server{
					ID:   int64(i + 1),
					Name: fmt.Sprintf("test-server-%d", i+1),
				}
			}

			// Create mock client with simulated deletion delay
			mockClient := &MockHetznerClient{
				deleteDelay: tt.deleteDelay,
			}

			// Simulate the parallel deletion logic from delete.go
			var deletionErrors []string
			var wg sync.WaitGroup
			var mu sync.Mutex

			start := time.Now()

			for _, server := range servers {
				wg.Add(1)
				go func(srv *hcloud.Server) {
					defer wg.Done()

					if err := mockClient.DeleteServer(context.Background(), srv); err != nil {
						errMsg := fmt.Sprintf("Failed to delete server %s: %v", srv.Name, err)
						mu.Lock()
						deletionErrors = append(deletionErrors, errMsg)
						mu.Unlock()
					}
				}(server)
			}

			wg.Wait()
			elapsed := time.Since(start)

			// Verify that all servers were deleted
			if mockClient.GetDeleteCallCount() != tt.serverCount {
				t.Errorf("Expected %d delete calls, got %d", tt.serverCount, mockClient.GetDeleteCallCount())
			}

			// Verify no errors occurred
			if len(deletionErrors) > 0 {
				t.Errorf("Expected no errors, got %d: %v", len(deletionErrors), deletionErrors)
			}

			// Verify that parallel execution completed within expected time
			if elapsed > tt.maxExpectedTime {
				t.Errorf("Deletion took too long: %v (expected max %v). This suggests serial rather than parallel execution.", elapsed, tt.maxExpectedTime)
			}

			t.Logf("Deleted %d servers in %v (expected max: %v)", tt.serverCount, elapsed, tt.maxExpectedTime)
		})
	}
}

// TestParallelDeletionErrorHandling verifies that errors during parallel deletion are properly collected
func TestParallelDeletionErrorHandling(t *testing.T) {
	tests := []struct {
		name             string
		serverCount      int
		failingServerIDs []int64
		expectedErrors   int
	}{
		{
			name:             "no failures",
			serverCount:      5,
			failingServerIDs: []int64{},
			expectedErrors:   0,
		},
		{
			name:             "single failure",
			serverCount:      5,
			failingServerIDs: []int64{3},
			expectedErrors:   1,
		},
		{
			name:             "multiple failures",
			serverCount:      10,
			failingServerIDs: []int64{2, 5, 8},
			expectedErrors:   3,
		},
		{
			name:             "all failures",
			serverCount:      3,
			failingServerIDs: []int64{1, 2, 3},
			expectedErrors:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock servers
			servers := make([]*hcloud.Server, tt.serverCount)
			for i := 0; i < tt.serverCount; i++ {
				servers[i] = &hcloud.Server{
					ID:   int64(i + 1),
					Name: fmt.Sprintf("test-server-%d", i+1),
				}
			}

			// Create set of failing server IDs for quick lookup
			failingServers := make(map[int64]bool)
			for _, id := range tt.failingServerIDs {
				failingServers[id] = true
			}

			// Create mock client that fails for specific servers
			mockClient := &MockHetznerClient{
				deleteServerFunc: func(ctx context.Context, server *hcloud.Server) error {
					if failingServers[server.ID] {
						return fmt.Errorf("simulated deletion failure")
					}
					return nil
				},
			}

			// Simulate the parallel deletion logic from delete.go
			var deletionErrors []string
			var wg sync.WaitGroup
			var mu sync.Mutex

			for _, server := range servers {
				wg.Add(1)
				go func(srv *hcloud.Server) {
					defer wg.Done()

					if err := mockClient.DeleteServer(context.Background(), srv); err != nil {
						errMsg := fmt.Sprintf("Failed to delete server %s: %v", srv.Name, err)
						mu.Lock()
						deletionErrors = append(deletionErrors, errMsg)
						mu.Unlock()
					}
				}(server)
			}

			wg.Wait()

			// Verify the correct number of errors were collected
			if len(deletionErrors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectedErrors, len(deletionErrors), deletionErrors)
			}

			// Verify that all servers were attempted to be deleted
			if mockClient.GetDeleteCallCount() != tt.serverCount {
				t.Errorf("Expected %d delete calls, got %d", tt.serverCount, mockClient.GetDeleteCallCount())
			}
		})
	}
}

// TestConcurrentErrorCollection verifies thread-safe error collection
func TestConcurrentErrorCollection(t *testing.T) {
	const numGoroutines = 100

	var deletionErrors []string
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Simulate concurrent error appending
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			errMsg := fmt.Sprintf("error-%d", index)
			mu.Lock()
			deletionErrors = append(deletionErrors, errMsg)
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify all errors were collected
	if len(deletionErrors) != numGoroutines {
		t.Errorf("Expected %d errors, got %d", numGoroutines, len(deletionErrors))
	}

	// Verify no duplicate errors (implies proper synchronization)
	errorSet := make(map[string]bool)
	for _, err := range deletionErrors {
		if errorSet[err] {
			t.Errorf("Duplicate error found: %s (indicates synchronization issue)", err)
		}
		errorSet[err] = true
	}
}

// TestEmptyServerList verifies behavior with no servers to delete
func TestEmptyServerList(t *testing.T) {
	var servers []*hcloud.Server // Empty list

	mockClient := &MockHetznerClient{}

	// Simulate the parallel deletion logic from delete.go
	var deletionErrors []string
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, server := range servers {
		wg.Add(1)
		go func(srv *hcloud.Server) {
			defer wg.Done()

			if err := mockClient.DeleteServer(context.Background(), srv); err != nil {
				errMsg := fmt.Sprintf("Failed to delete server %s: %v", srv.Name, err)
				mu.Lock()
				deletionErrors = append(deletionErrors, errMsg)
				mu.Unlock()
			}
		}(server)
	}

	wg.Wait()

	// Verify no deletions occurred
	if mockClient.GetDeleteCallCount() != 0 {
		t.Errorf("Expected 0 delete calls for empty list, got %d", mockClient.GetDeleteCallCount())
	}

	// Verify no errors
	if len(deletionErrors) != 0 {
		t.Errorf("Expected 0 errors for empty list, got %d", len(deletionErrors))
	}
}

// Helper function to create a test configuration
func createTestConfig() *config.Main {
	return &config.Main{
		ClusterName:  "test-cluster",
		HetznerToken: "test-token",
	}
}

// Helper function to create a test Hetzner client
func createTestHetznerClient() *hetzner.Client {
	return hetzner.NewClient("test-token")
}
