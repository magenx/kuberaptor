package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteToFile_CreatesParentDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Test writing to a file in a non-existent subdirectory
	testPath := filepath.Join(tmpDir, "subdir", "nested", "config")
	testData := []byte("test kubeconfig content")

	// Write file - should create parent directories
	err := WriteToFile(testPath, testData, 0600)
	if err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	// Verify the file was created
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Errorf("File was not created at %s", testPath)
	}

	// Verify the content is correct
	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != string(testData) {
		t.Errorf("File content mismatch. Expected: %s, Got: %s", testData, content)
	}

	// Verify file permissions
	info, err := os.Stat(testPath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("File permissions mismatch. Expected: 0600, Got: %o", info.Mode().Perm())
	}
}

func TestWriteToFile_ExistingDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Test writing to a file in an existing directory
	testPath := filepath.Join(tmpDir, "config")
	testData := []byte("test content")

	// Write file - directory already exists
	err := WriteToFile(testPath, testData, 0644)
	if err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	// Verify the file was created
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Errorf("File was not created at %s", testPath)
	}

	// Verify the content is correct
	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != string(testData) {
		t.Errorf("File content mismatch. Expected: %s, Got: %s", testData, content)
	}
}

func TestWriteToFile_OverwriteExisting(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Test overwriting an existing file
	testPath := filepath.Join(tmpDir, "config")
	initialData := []byte("initial content")
	newData := []byte("new content")

	// Write initial file
	err := WriteToFile(testPath, initialData, 0600)
	if err != nil {
		t.Fatalf("Initial WriteToFile failed: %v", err)
	}

	// Overwrite with new data
	err = WriteToFile(testPath, newData, 0600)
	if err != nil {
		t.Fatalf("WriteToFile overwrite failed: %v", err)
	}

	// Verify the content was updated
	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != string(newData) {
		t.Errorf("File content mismatch. Expected: %s, Got: %s", newData, content)
	}
}

func TestWriteToFile_KubeconfigScenario(t *testing.T) {
	// Simulate the exact scenario from the issue
	tmpDir := t.TempDir()

	// Simulate writing to ~/.kube/config (which doesn't exist yet)
	kubeconfigPath := filepath.Join(tmpDir, ".kube", "config")
	kubeconfigContent := []byte(`apiVersion: v1
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: default
`)

	// Write kubeconfig - should create .kube directory
	err := WriteToFile(kubeconfigPath, kubeconfigContent, 0600)
	if err != nil {
		t.Fatalf("WriteToFile failed for kubeconfig: %v", err)
	}

	// Verify the .kube directory was created
	kubeDir := filepath.Join(tmpDir, ".kube")
	if _, err := os.Stat(kubeDir); os.IsNotExist(err) {
		t.Errorf(".kube directory was not created at %s", kubeDir)
	}

	// Verify directory permissions (should be 0755)
	dirInfo, err := os.Stat(kubeDir)
	if err != nil {
		t.Fatalf("Failed to stat .kube directory: %v", err)
	}
	if dirInfo.Mode().Perm() != 0755 {
		t.Errorf("Directory permissions mismatch. Expected: 0755, Got: %o", dirInfo.Mode().Perm())
	}

	// Verify the kubeconfig file was created
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		t.Errorf("Kubeconfig file was not created at %s", kubeconfigPath)
	}

	// Verify kubeconfig content
	content, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		t.Fatalf("Failed to read kubeconfig: %v", err)
	}
	if string(content) != string(kubeconfigContent) {
		t.Errorf("Kubeconfig content mismatch.")
	}
}
