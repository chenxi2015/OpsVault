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

// QdrantDriver represents the Docker driver for Qdrant vector database service
type QdrantDriver struct {
	*BaseDriver
	apiKey string
}

// NewQdrantDriver creates and returns a new QdrantDriver instance
func NewQdrantDriver(cli DockerClient, cfg *viper.Viper, apiKey string) *QdrantDriver {
	port := cfg.GetInt("qdrant.port")
	if port == 0 {
		port = 6333
	}
	grpcPort := cfg.GetInt("qdrant.grpc_port")
	if grpcPort == 0 {
		grpcPort = 6334
	}
	image := cfg.GetString("qdrant.image")
	if image == "" {
		image = "qdrant/qdrant:v1.13.0"
	}
	if apiKey == "" {
		apiKey = cfg.GetString("qdrant.api_key")
	}

	base := NewBaseDriver("qdrant", cli.Raw(), cfg, image, []string{
		fmt.Sprintf("%d:6333", port),
		fmt.Sprintf("%d:6334", grpcPort),
	})
	drv := &QdrantDriver{BaseDriver: base, apiKey: apiKey}
	drv.PrepareConfig = drv.prepareConfig
	return drv
}

// Install runs the installer for Qdrant container
func (d *QdrantDriver) Install() error {
	return d.installWithSpec(d.containerSpec)
}

func (d *QdrantDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("6333/tcp")
	grpcPort := nat.Port("6334/tcp")

	hostPort := d.Config.GetString("qdrant.port")
	if hostPort == "" {
		hostPort = "6333"
	}
	hostGrpcPort := d.Config.GetString("qdrant.grpc_port")
	if hostGrpcPort == "" {
		hostGrpcPort = "6334"
	}

	env := []string{}
	if d.apiKey != "" {
		env = append(env, "QDRANT__SERVICE__API_KEY="+d.apiKey)
	}

	storagePath := filepath.Join(d.DataDir, "storage")

	return &container.Config{
			Image: d.Image,
			Env:   env,
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD-SHELL", "curl -f http://localhost:6333/healthz || exit 1"},
				Interval:    10 * time.Second,
				Timeout:     5 * time.Second,
				StartPeriod: 10 * time.Second,
				Retries:     5,
			},
		}, &container.HostConfig{
			Binds: []string{
				toDockerBind(storagePath, "/qdrant/storage"),
			},
			PortBindings: nat.PortMap{
				port:     []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostPort}},
				grpcPort: []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostGrpcPort}},
			},
		}, nil
}

func (d *QdrantDriver) prepareConfig(confDir string) error {
	storagePath := filepath.Join(d.DataDir, "storage")
	return fileutil.EnsureDir(storagePath, 0755)
}

// Upgrade upgrades the Qdrant service to target version
func (d *QdrantDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}

// GetCredentials returns connection credentials for Qdrant
func (d *QdrantDriver) GetCredentials() []credutil.Credential {
	port := d.Config.GetString("qdrant.port")
	if port == "" {
		port = "6333"
	}
	grpcPort := d.Config.GetString("qdrant.grpc_port")
	if grpcPort == "" {
		grpcPort = "6334"
	}
	creds := []credutil.Credential{
		{Label: "HTTP API & Web Dashboard", Value: fmt.Sprintf("http://localhost:%s", port)},
		{Label: "gRPC Address", Value: fmt.Sprintf("localhost:%s", grpcPort)},
	}
	if d.apiKey != "" {
		creds = append(creds, credutil.Credential{Label: "API Key", Value: d.apiKey})
	}
	return creds
}
