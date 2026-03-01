package config

import (
	"fmt"
	"strings"
)

// Datastore represents datastore configuration
type Datastore struct {
	Mode              string             `yaml:"mode,omitempty"`
	ExternalDatastore *ExternalDatastore `yaml:"external_datastore,omitempty"`
	EmbeddedEtcd      *EmbeddedEtcd      `yaml:"embedded_etcd,omitempty"`
}

// SetDefaults sets default values for datastore
func (d *Datastore) SetDefaults() {
	if d.Mode == "" {
		d.Mode = "etcd"
	}
	if d.EmbeddedEtcd != nil {
		d.EmbeddedEtcd.SetDefaults()
	}
}

// ExternalDatastore represents external datastore configuration
type ExternalDatastore struct {
	Endpoint string `yaml:"endpoint"`
	CaFile   string `yaml:"ca_file,omitempty"`
	CertFile string `yaml:"cert_file,omitempty"`
	KeyFile  string `yaml:"key_file,omitempty"`
}

// EmbeddedEtcd represents embedded etcd configuration
type EmbeddedEtcd struct {
	SnapshotRetention    int64  `yaml:"snapshot_retention,omitempty"`
	SnapshotScheduleCron string `yaml:"snapshot_schedule_cron,omitempty"`
	S3Enabled            bool   `yaml:"s3_enabled,omitempty"`
	S3Endpoint           string `yaml:"s3_endpoint,omitempty"`
	S3Region             string `yaml:"s3_region,omitempty"`
	S3Bucket             string `yaml:"s3_bucket,omitempty"`
	S3Folder             string `yaml:"s3_folder,omitempty"`
	S3AccessKey          string `yaml:"s3_access_key,omitempty"`
	S3SecretKey          string `yaml:"s3_secret_key,omitempty"`
	S3ForcePathStyle     bool   `yaml:"s3_force_path_style,omitempty"`
}

// Addons represents addon configuration
type Addons struct {
	Traefik                 *Toggle                  `yaml:"traefik,omitempty"`
	ServiceLB               *Toggle                  `yaml:"servicelb,omitempty"`
	MetricsServer           *Toggle                  `yaml:"metrics_server,omitempty"`
	EmbeddedRegistryMirror  *Toggle                  `yaml:"embedded_registry_mirror,omitempty"`
	LocalPathStorageClass   *Toggle                  `yaml:"local_path_storage_class,omitempty"`
	CSIDriver               *CSIDriver               `yaml:"csi_driver,omitempty"`
	ClusterAutoscaler       *ClusterAutoscaler       `yaml:"cluster_autoscaler,omitempty"`
	CloudControllerManager  *CloudControllerManager  `yaml:"cloud_controller_manager,omitempty"`
	SystemUpgradeController *SystemUpgradeController `yaml:"system_upgrade_controller,omitempty"`
}

// SetDefaults sets default values for addons
func (a *Addons) SetDefaults() {
	if a.Traefik == nil {
		a.Traefik = &Toggle{Enabled: false}
	}
	if a.ServiceLB == nil {
		a.ServiceLB = &Toggle{Enabled: false}
	}
	if a.MetricsServer == nil {
		a.MetricsServer = &Toggle{Enabled: false}
	}
	if a.EmbeddedRegistryMirror == nil {
		a.EmbeddedRegistryMirror = &Toggle{Enabled: true}
	}
	if a.LocalPathStorageClass == nil {
		a.LocalPathStorageClass = &Toggle{Enabled: false}
	}
	if a.CSIDriver == nil {
		a.CSIDriver = &CSIDriver{}
	}
	a.CSIDriver.SetDefaults()
	if a.ClusterAutoscaler == nil {
		a.ClusterAutoscaler = &ClusterAutoscaler{}
	}
	a.ClusterAutoscaler.SetDefaults()
	if a.CloudControllerManager == nil {
		a.CloudControllerManager = &CloudControllerManager{}
	}
	a.CloudControllerManager.SetDefaults()
	if a.SystemUpgradeController == nil {
		a.SystemUpgradeController = &SystemUpgradeController{}
	}
	a.SystemUpgradeController.SetDefaults()
}

// Toggle represents a simple enabled/disabled toggle
type Toggle struct {
	Enabled bool `yaml:"enabled,omitempty"`
}

// CSIDriver represents CSI driver configuration
type CSIDriver struct {
	Enabled     bool   `yaml:"enabled,omitempty"`
	Version     string `yaml:"version,omitempty"`
	ManifestURL string `yaml:"manifest_url,omitempty"`
}

// SetDefaults sets default values for CSI driver
func (c *CSIDriver) SetDefaults() {
	if !c.Enabled {
		c.Enabled = true
	}
	if c.ManifestURL == "" {
		c.ManifestURL = "https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.18.3/deploy/kubernetes/hcloud-csi.yml"
	}
}

// ClusterAutoscaler represents cluster autoscaler configuration
type ClusterAutoscaler struct {
	Enabled                    bool   `yaml:"enabled,omitempty"`
	Version                    string `yaml:"version,omitempty"`
	ManifestURL                string `yaml:"manifest_url,omitempty"`
	ContainerImageTag          string `yaml:"container_image_tag,omitempty"`
	ScanInterval               string `yaml:"scan_interval,omitempty"`
	ScaleDownDelayAfterAdd     string `yaml:"scale_down_delay_after_add,omitempty"`
	ScaleDownDelayAfterDelete  string `yaml:"scale_down_delay_after_delete,omitempty"`
	ScaleDownDelayAfterFailure string `yaml:"scale_down_delay_after_failure,omitempty"`
	MaxNodeProvisionTime       string `yaml:"max_node_provision_time,omitempty"`
}

// SetDefaults sets default values for cluster autoscaler
func (c *ClusterAutoscaler) SetDefaults() {
	if !c.Enabled {
		c.Enabled = true
	}
	if c.ManifestURL == "" {
		c.ManifestURL = "https://raw.githubusercontent.com/kubernetes/autoscaler/master/cluster-autoscaler/cloudprovider/hetzner/examples/cluster-autoscaler-run-on-master.yaml"
	}
	if c.ContainerImageTag == "" {
		c.ContainerImageTag = "v1.34.2"
	}
	if c.ScanInterval == "" {
		c.ScanInterval = "10s"
	}
	if c.ScaleDownDelayAfterAdd == "" {
		c.ScaleDownDelayAfterAdd = "10m"
	}
	if c.ScaleDownDelayAfterDelete == "" {
		c.ScaleDownDelayAfterDelete = "10s"
	}
	if c.ScaleDownDelayAfterFailure == "" {
		c.ScaleDownDelayAfterFailure = "3m"
	}
	if c.MaxNodeProvisionTime == "" {
		c.MaxNodeProvisionTime = "15m"
	}
}

// CloudControllerManager represents cloud controller manager configuration
type CloudControllerManager struct {
	Enabled     bool   `yaml:"enabled,omitempty"`
	Version     string `yaml:"version,omitempty"`
	ManifestURL string `yaml:"manifest_url,omitempty"`
}

// SetDefaults sets default values for cloud controller manager
func (c *CloudControllerManager) SetDefaults() {
	if !c.Enabled {
		c.Enabled = true
	}
	if c.ManifestURL == "" {
		c.ManifestURL = "https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/download/v1.28.0/ccm-networks.yaml"
	}
}

// SystemUpgradeController represents system upgrade controller configuration
type SystemUpgradeController struct {
	Enabled               bool   `yaml:"enabled,omitempty"`
	Version               string `yaml:"version,omitempty"`
	DeploymentManifestURL string `yaml:"deployment_manifest_url,omitempty"`
	CRDManifestURL        string `yaml:"crd_manifest_url,omitempty"`
}

// SetDefaults sets default values for system upgrade controller
func (s *SystemUpgradeController) SetDefaults() {
	if !s.Enabled {
		s.Enabled = true
	}
	if s.DeploymentManifestURL == "" {
		s.DeploymentManifestURL = "https://github.com/rancher/system-upgrade-controller/releases/download/v0.18.0/system-upgrade-controller.yaml"
	}
	if s.CRDManifestURL == "" {
		s.CRDManifestURL = "https://github.com/rancher/system-upgrade-controller/releases/download/v0.18.0/crd.yaml"
	}
}

// SetDefaults sets default values for embedded etcd
func (e *EmbeddedEtcd) SetDefaults() {
	if e.SnapshotRetention == 0 {
		e.SnapshotRetention = 24
	}
	if e.SnapshotScheduleCron == "" {
		e.SnapshotScheduleCron = "0 * * * *"
	}
}

// IsS3Configured checks if S3 is properly configured
func (e *EmbeddedEtcd) IsS3Configured() bool {
	if !e.S3Enabled {
		return false
	}
	return e.S3Endpoint != "" && e.S3Region != "" && e.S3Bucket != "" &&
		e.S3AccessKey != "" && e.S3SecretKey != ""
}

// GenerateEtcdArgs generates etcd command-line arguments for k3s
func (e *EmbeddedEtcd) GenerateEtcdArgs() string {
	if e == nil {
		return ""
	}

	var args []string

	if e.SnapshotRetention > 0 {
		args = append(args, fmt.Sprintf("--etcd-snapshot-retention=%d", e.SnapshotRetention))
	}

	if e.SnapshotScheduleCron != "" {
		args = append(args, fmt.Sprintf("--etcd-snapshot-schedule-cron='%s'", e.SnapshotScheduleCron))
	}

	if e.IsS3Configured() {
		args = append(args, "--etcd-s3")
		args = append(args, fmt.Sprintf("--etcd-s3-endpoint=%s", e.S3Endpoint))
		args = append(args, fmt.Sprintf("--etcd-s3-region=%s", e.S3Region))
		args = append(args, fmt.Sprintf("--etcd-s3-bucket=%s", e.S3Bucket))
		args = append(args, fmt.Sprintf("--etcd-s3-access-key=%s", e.S3AccessKey))
		args = append(args, fmt.Sprintf("--etcd-s3-secret-key=%s", e.S3SecretKey))

		if e.S3ForcePathStyle {
			args = append(args, "--etcd-s3-force-path-style")
		}

		if e.S3Folder != "" {
			args = append(args, fmt.Sprintf("--etcd-s3-folder=%s", e.S3Folder))
		}
	}

	return strings.Join(args, " ")
}
