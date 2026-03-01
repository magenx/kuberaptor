package config

import (
	"strings"
	"testing"
)

func TestEmbeddedEtcd_SetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    *EmbeddedEtcd
		wantRet  int64
		wantCron string
	}{
		{
			name:     "empty config gets defaults",
			input:    &EmbeddedEtcd{},
			wantRet:  24,
			wantCron: "0 * * * *",
		},
		{
			name: "existing values are preserved",
			input: &EmbeddedEtcd{
				SnapshotRetention:    48,
				SnapshotScheduleCron: "0 0 * * *",
			},
			wantRet:  48,
			wantCron: "0 0 * * *",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.SetDefaults()
			if tt.input.SnapshotRetention != tt.wantRet {
				t.Errorf("SnapshotRetention = %d, want %d", tt.input.SnapshotRetention, tt.wantRet)
			}
			if tt.input.SnapshotScheduleCron != tt.wantCron {
				t.Errorf("SnapshotScheduleCron = %s, want %s", tt.input.SnapshotScheduleCron, tt.wantCron)
			}
		})
	}
}

func TestEmbeddedEtcd_IsS3Configured(t *testing.T) {
	tests := []struct {
		name  string
		input *EmbeddedEtcd
		want  bool
	}{
		{
			name: "fully configured S3",
			input: &EmbeddedEtcd{
				S3Enabled:   true,
				S3Endpoint:  "https://fsn1.your-objectstorage.com",
				S3Region:    "fsn1",
				S3Bucket:    "my-bucket",
				S3AccessKey: "access-key",
				S3SecretKey: "secret-key",
			},
			want: true,
		},
		{
			name: "S3 disabled",
			input: &EmbeddedEtcd{
				S3Enabled:   false,
				S3Endpoint:  "https://fsn1.your-objectstorage.com",
				S3Region:    "fsn1",
				S3Bucket:    "my-bucket",
				S3AccessKey: "access-key",
				S3SecretKey: "secret-key",
			},
			want: false,
		},
		{
			name: "missing endpoint",
			input: &EmbeddedEtcd{
				S3Enabled:   true,
				S3Region:    "fsn1",
				S3Bucket:    "my-bucket",
				S3AccessKey: "access-key",
				S3SecretKey: "secret-key",
			},
			want: false,
		},
		{
			name: "missing region",
			input: &EmbeddedEtcd{
				S3Enabled:   true,
				S3Endpoint:  "https://fsn1.your-objectstorage.com",
				S3Bucket:    "my-bucket",
				S3AccessKey: "access-key",
				S3SecretKey: "secret-key",
			},
			want: false,
		},
		{
			name: "missing bucket",
			input: &EmbeddedEtcd{
				S3Enabled:   true,
				S3Endpoint:  "https://fsn1.your-objectstorage.com",
				S3Region:    "fsn1",
				S3AccessKey: "access-key",
				S3SecretKey: "secret-key",
			},
			want: false,
		},
		{
			name: "missing credentials",
			input: &EmbeddedEtcd{
				S3Enabled:  true,
				S3Endpoint: "https://fsn1.your-objectstorage.com",
				S3Region:   "fsn1",
				S3Bucket:   "my-bucket",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.IsS3Configured()
			if got != tt.want {
				t.Errorf("IsS3Configured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmbeddedEtcd_GenerateEtcdArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    *EmbeddedEtcd
		wantArgs []string // Expected arguments to be present
	}{
		{
			name:     "nil config returns empty string",
			input:    nil,
			wantArgs: []string{},
		},
		{
			name: "basic snapshot configuration",
			input: &EmbeddedEtcd{
				SnapshotRetention:    24,
				SnapshotScheduleCron: "0 * * * *",
			},
			wantArgs: []string{
				"--etcd-snapshot-retention=24",
				"--etcd-snapshot-schedule-cron='0 * * * *'",
			},
		},
		{
			name: "full S3 configuration",
			input: &EmbeddedEtcd{
				SnapshotRetention:    48,
				SnapshotScheduleCron: "0 0 * * *",
				S3Enabled:            true,
				S3Endpoint:           "fsn1.your-objectstorage.com",
				S3Region:             "fsn1",
				S3Bucket:             "my-bucket",
				S3AccessKey:          "access-key",
				S3SecretKey:          "secret-key",
				S3Folder:             "etcd-backups",
				S3ForcePathStyle:     true,
			},
			wantArgs: []string{
				"--etcd-snapshot-retention=48",
				"--etcd-snapshot-schedule-cron='0 0 * * *'",
				"--etcd-s3",
				"--etcd-s3-endpoint=fsn1.your-objectstorage.com",
				"--etcd-s3-region=fsn1",
				"--etcd-s3-bucket=my-bucket",
				"--etcd-s3-access-key=access-key",
				"--etcd-s3-secret-key=secret-key",
				"--etcd-s3-force-path-style",
				"--etcd-s3-folder=etcd-backups",
			},
		},
		{
			name: "S3 without folder",
			input: &EmbeddedEtcd{
				SnapshotRetention:    24,
				SnapshotScheduleCron: "0 * * * *",
				S3Enabled:            true,
				S3Endpoint:           "fsn1.your-objectstorage.com",
				S3Region:             "fsn1",
				S3Bucket:             "my-bucket",
				S3AccessKey:          "access-key",
				S3SecretKey:          "secret-key",
			},
			wantArgs: []string{
				"--etcd-s3",
				"--etcd-s3-endpoint=fsn1.your-objectstorage.com",
				"--etcd-s3-region=fsn1",
				"--etcd-s3-bucket=my-bucket",
			},
		},
		{
			name: "S3 not configured (incomplete)",
			input: &EmbeddedEtcd{
				SnapshotRetention:    24,
				SnapshotScheduleCron: "0 * * * *",
				S3Enabled:            true,
				S3Endpoint:           "fsn1.your-objectstorage.com",
				// Missing other S3 fields
			},
			wantArgs: []string{
				"--etcd-snapshot-retention=24",
				"--etcd-snapshot-schedule-cron='0 * * * *'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.GenerateEtcdArgs()

			if len(tt.wantArgs) == 0 && got != "" {
				t.Errorf("GenerateEtcdArgs() = %q, want empty string", got)
				return
			}

			for _, wantArg := range tt.wantArgs {
				if !strings.Contains(got, wantArg) {
					t.Errorf("GenerateEtcdArgs() missing argument %q, got: %q", wantArg, got)
				}
			}

			// Check that unwanted args are not present
			if tt.name == "S3 not configured (incomplete)" {
				if strings.Contains(got, "--etcd-s3") {
					t.Errorf("GenerateEtcdArgs() should not contain --etcd-s3 for incomplete config, got: %q", got)
				}
			}
		})
	}
}

func TestDatastore_SetDefaults_CallsEmbeddedEtcdSetDefaults(t *testing.T) {
	ds := &Datastore{
		EmbeddedEtcd: &EmbeddedEtcd{},
	}

	ds.SetDefaults()

	if ds.Mode != "etcd" {
		t.Errorf("Mode = %s, want etcd", ds.Mode)
	}

	if ds.EmbeddedEtcd.SnapshotRetention != 24 {
		t.Errorf("EmbeddedEtcd.SnapshotRetention = %d, want 24", ds.EmbeddedEtcd.SnapshotRetention)
	}

	if ds.EmbeddedEtcd.SnapshotScheduleCron != "0 * * * *" {
		t.Errorf("EmbeddedEtcd.SnapshotScheduleCron = %s, want '0 * * * *'", ds.EmbeddedEtcd.SnapshotScheduleCron)
	}
}
