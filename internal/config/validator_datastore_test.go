package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func getValidTestConfig(t *testing.T) *Main {
	// Create temporary SSH keys for testing using t.TempDir() for automatic cleanup
	tmpDir := t.TempDir()

	privateKeyPath := filepath.Join(tmpDir, "id_rsa")
	publicKeyPath := filepath.Join(tmpDir, "id_rsa.pub")

	// Create dummy SSH key files
	if err := os.WriteFile(privateKeyPath, []byte("dummy private key"), 0600); err != nil {
		t.Fatalf("Failed to create test private key: %v", err)
	}
	if err := os.WriteFile(publicKeyPath, []byte("dummy public key"), 0644); err != nil {
		t.Fatalf("Failed to create test public key: %v", err)
	}

	return &Main{
		ClusterName: "test-cluster",
		K3sVersion:  "v1.32.0+k3s1",
		MastersPool: MasterNodePool{
			NodePool: NodePool{
				InstanceType:  "cx11",
				InstanceCount: 1,
			},
			Locations: []string{"fsn1"},
		},
		Networking: Networking{
			SSH: SSH{
				Port:           22,
				PrivateKeyPath: privateKeyPath,
				PublicKeyPath:  publicKeyPath,
			},
			AllowedNetworks: AllowedNetworks{
				SSH: []string{"0.0.0.0/0"},
				API: []string{"0.0.0.0/0"},
			},
		},
	}
}

func TestValidateDatastore_ValidEtcdWithoutS3(t *testing.T) {
	cfg := getValidTestConfig(t)
	cfg.Datastore = Datastore{
		Mode: "etcd",
		EmbeddedEtcd: &EmbeddedEtcd{
			SnapshotRetention:    24,
			SnapshotScheduleCron: "0 * * * *",
			S3Enabled:            false,
		},
	}

	validator := NewValidator(cfg)
	err := validator.Validate()
	if err != nil {
		t.Errorf("Expected valid configuration, got error: %v", err)
	}
}

func TestValidateDatastore_ValidEtcdWithS3(t *testing.T) {
	cfg := getValidTestConfig(t)
	cfg.Datastore = Datastore{
		Mode: "etcd",
		EmbeddedEtcd: &EmbeddedEtcd{
			SnapshotRetention:    24,
			SnapshotScheduleCron: "0 * * * *",
			S3Enabled:            true,
			S3Endpoint:           "https://fsn1.your-objectstorage.com",
			S3Region:             "fsn1",
			S3Bucket:             "my-bucket",
			S3AccessKey:          "access-key",
			S3SecretKey:          "secret-key",
		},
	}

	validator := NewValidator(cfg)
	err := validator.Validate()
	if err != nil {
		t.Errorf("Expected valid configuration, got error: %v", err)
	}

	// Should have warning about bucket existing
	hasWarning := false
	for _, warning := range validator.warnings {
		if strings.Contains(warning, "ensure it exists before cluster creation") {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("Expected warning about bucket existing before cluster creation")
	}
}

func TestValidateDatastore_S3EnabledMissingEndpoint(t *testing.T) {
	cfg := getValidTestConfig(t)
	cfg.Datastore = Datastore{
		Mode: "etcd",
		EmbeddedEtcd: &EmbeddedEtcd{
			S3Enabled:   true,
			S3Region:    "fsn1",
			S3Bucket:    "my-bucket",
			S3AccessKey: "access-key",
			S3SecretKey: "secret-key",
		},
	}

	validator := NewValidator(cfg)
	err := validator.Validate()
	if err == nil {
		t.Error("Expected error for missing s3_endpoint, got nil")
	}
	if !strings.Contains(err.Error(), "s3_endpoint is required") {
		t.Errorf("Expected error about missing s3_endpoint, got: %v", err)
	}
}

func TestValidateDatastore_S3EnabledMissingCredentials(t *testing.T) {
	cfg := getValidTestConfig(t)
	cfg.Datastore = Datastore{
		Mode: "etcd",
		EmbeddedEtcd: &EmbeddedEtcd{
			S3Enabled:  true,
			S3Endpoint: "https://fsn1.your-objectstorage.com",
			S3Region:   "fsn1",
			S3Bucket:   "my-bucket",
			// Missing credentials
		},
	}

	validator := NewValidator(cfg)
	err := validator.Validate()
	if err == nil {
		t.Error("Expected error for missing S3 credentials, got nil")
	}
	if !strings.Contains(err.Error(), "s3_access_key is required") {
		t.Errorf("Expected error about missing s3_access_key, got: %v", err)
	}
}

func TestValidateDatastore_S3EnabledWithoutBucketName(t *testing.T) {
	cfg := getValidTestConfig(t)
	cfg.Datastore = Datastore{
		Mode: "etcd",
		EmbeddedEtcd: &EmbeddedEtcd{
			S3Enabled:   true,
			S3Endpoint:  "https://fsn1.your-objectstorage.com",
			S3Region:    "fsn1",
			S3AccessKey: "access-key",
			S3SecretKey: "secret-key",
			// No bucket name - should error
		},
	}

	validator := NewValidator(cfg)
	err := validator.Validate()
	if err == nil {
		t.Error("Expected error for missing bucket name when S3 is enabled, got nil")
	}
	if !strings.Contains(err.Error(), "s3_bucket is required") {
		t.Errorf("Expected error about missing s3_bucket, got: %v", err)
	}
}

func TestValidateDatastore_InvalidMode(t *testing.T) {
	cfg := getValidTestConfig(t)
	cfg.Datastore = Datastore{
		Mode: "invalid-mode",
	}

	validator := NewValidator(cfg)
	err := validator.Validate()
	if err == nil {
		t.Error("Expected error for invalid datastore mode, got nil")
	}
	if !strings.Contains(err.Error(), "invalid mode") {
		t.Errorf("Expected error about invalid mode, got: %v", err)
	}
}

func TestValidateDatastore_NegativeSnapshotRetention(t *testing.T) {
	cfg := getValidTestConfig(t)
	cfg.Datastore = Datastore{
		Mode: "etcd",
		EmbeddedEtcd: &EmbeddedEtcd{
			SnapshotRetention: -1,
		},
	}

	validator := NewValidator(cfg)
	err := validator.Validate()
	if err == nil {
		t.Error("Expected error for negative snapshot retention, got nil")
	}
	if !strings.Contains(err.Error(), "snapshot_retention cannot be negative") {
		t.Errorf("Expected error about negative snapshot_retention, got: %v", err)
	}
}

func TestValidateDatastore_ExternalDatastoreMissingEndpoint(t *testing.T) {
	cfg := getValidTestConfig(t)
	cfg.Datastore = Datastore{
		Mode:              "external",
		ExternalDatastore: &ExternalDatastore{},
	}

	validator := NewValidator(cfg)
	err := validator.Validate()
	if err == nil {
		t.Error("Expected error for missing external datastore endpoint, got nil")
	}
	if !strings.Contains(err.Error(), "endpoint is required") {
		t.Errorf("Expected error about missing endpoint, got: %v", err)
	}
}
