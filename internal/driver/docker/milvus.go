package docker

import (
	"fmt"
	"path/filepath"
	"time"

	"OpsVault/pkg/credutil"
	"OpsVault/pkg/fileutil"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/viper"
)

// MilvusDriver represents the Docker driver for Milvus vector database service
type MilvusDriver struct {
	*BaseDriver
}

// NewMilvusDriver creates and returns a new MilvusDriver instance
func NewMilvusDriver(cli DockerClient, cfg *viper.Viper) *MilvusDriver {
	port := cfg.GetInt("milvus.port")
	if port == 0 {
		port = 19530
	}
	metricsPort := cfg.GetInt("milvus.metrics_port")
	if metricsPort == 0 {
		metricsPort = 9091
	}
	image := cfg.GetString("milvus.image")
	if image == "" {
		image = "milvusdb/milvus:v2.4.15"
	}

	base := NewBaseDriver("milvus", cli.Raw(), cfg, image, []string{
		fmt.Sprintf("%d:19530", port),
		fmt.Sprintf("%d:9091", metricsPort),
	})
	drv := &MilvusDriver{BaseDriver: base}
	drv.PrepareConfig = drv.prepareConfig
	return drv
}

// Install runs the installer for Milvus container
func (d *MilvusDriver) Install() error {
	return d.installWithSpec(d.containerSpec)
}

func (d *MilvusDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("19530/tcp")
	metricsPort := nat.Port("9091/tcp")

	hostPort := d.Config.GetString("milvus.port")
	if hostPort == "" {
		hostPort = "19530"
	}
	hostMetricsPort := d.Config.GetString("milvus.metrics_port")
	if hostMetricsPort == "" {
		hostMetricsPort = "9091"
	}

	dataPath := filepath.Join(d.DataDir, "data")

	return &container.Config{
			Image: d.Image,
			Cmd:   []string{"milvus", "run", "standalone"},
			Env: []string{
				"ETCD_USE_EMBED=true",
				"ETCD_DATA_DIR=/var/lib/milvus/etcd",
				"COMMON_STORAGETYPE=local",
			},
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD-SHELL", "curl -f http://localhost:9091/healthz || exit 1"},
				Interval:    10 * time.Second,
				Timeout:     5 * time.Second,
				StartPeriod: 15 * time.Second,
				Retries:     5,
			},
		}, &container.HostConfig{
			Binds: []string{
				toDockerBind(dataPath, "/var/lib/milvus"),
			},
			PortBindings: nat.PortMap{
				port:        []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostPort}},
				metricsPort: []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostMetricsPort}},
			},
		}, nil
}

func (d *MilvusDriver) prepareConfig(confDir string) error {
	dataPath := filepath.Join(d.DataDir, "data")
	return fileutil.EnsureDir(dataPath, 0755)
}

// Upgrade upgrades the Milvus service to target version
func (d *MilvusDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}

// GetCredentials returns connection credentials for Milvus
func (d *MilvusDriver) GetCredentials() []credutil.Credential {
	port := d.Config.GetString("milvus.port")
	if port == "" {
		port = "19530"
	}
	metricsPort := d.Config.GetString("milvus.metrics_port")
	if metricsPort == "" {
		metricsPort = "9091"
	}
	return []credutil.Credential{
		{Label: "gRPC & RESTful API Endpoint", Value: fmt.Sprintf("localhost:%s", port)},
		{Label: "Metrics & Health Endpoint", Value: fmt.Sprintf("http://localhost:%s/healthz", metricsPort)},
	}
}

