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

// MinIODriver represents the Docker driver for MinIO service
type MinIODriver struct {
	*BaseDriver
	rootUser     string
	rootPassword string
}

// NewMinIODriver creates and returns a new MinIODriver instance
func NewMinIODriver(cli DockerClient, cfg *viper.Viper, rootPassword string) *MinIODriver {
	port := cfg.GetInt("minio.port")
	if port == 0 {
		port = 9000
	}
	consolePort := cfg.GetInt("minio.console_port")
	if consolePort == 0 {
		consolePort = 9001
	}
	image := cfg.GetString("minio.image")
	if image == "" {
		image = "minio/minio:RELEASE.2024-05-10T01-39-39Z"
	}
	rootUser := cfg.GetString("minio.root_user")
	if rootUser == "" {
		rootUser = "minioadmin"
	}
	if rootPassword == "" {
		rootPassword = cfg.GetString("minio.root_password")
	}

	base := NewBaseDriver("minio", cli.Raw(), cfg, image, []string{
		fmt.Sprintf("%d:9000", port),
		fmt.Sprintf("%d:9001", consolePort),
	})
	drv := &MinIODriver{BaseDriver: base, rootUser: rootUser, rootPassword: rootPassword}
	drv.PrepareConfig = drv.prepareConfig
	return drv
}

// Install runs the installer for MinIO container
func (d *MinIODriver) Install() error {
	if d.rootPassword == "" {
		pwd := credutil.GenPassword(20)
		d.rootPassword = pwd
		d.Config.Set("minio.root_password", pwd)
		cfgPath := d.Config.ConfigFileUsed()
		if cfgPath == "" {
			cfgPath = fileutil.GetDefaultWriteConfigPath()
		}
		_ = fileutil.UpdateYAMLValue(cfgPath, "minio", "root_password", pwd)
	}
	return d.installWithSpec(d.containerSpec)
}

func (d *MinIODriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("9000/tcp")
	consolePort := nat.Port("9001/tcp")

	hostPort := d.Config.GetString("minio.port")
	if hostPort == "" {
		hostPort = "9000"
	}
	hostConsolePort := d.Config.GetString("minio.console_port")
	if hostConsolePort == "" {
		hostConsolePort = "9001"
	}

	return &container.Config{
			Image: d.Image,
			Env: []string{
				"MINIO_ROOT_USER=" + d.rootUser,
				"MINIO_ROOT_PASSWORD=" + d.rootPassword,
			},
			Cmd: []string{"server", "/data", "--console-address", ":9001"},
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD", "mc", "ready", "local"},
				Interval:    10 * time.Second,
				Timeout:     5 * time.Second,
				StartPeriod: 15 * time.Second,
				Retries:     10,
			},
		}, &container.HostConfig{
			Binds: []string{
				toDockerBind(filepath.Join(d.DataDir, "data"), "/data"),
			},
			PortBindings: nat.PortMap{
				port:        []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostPort}},
				consolePort: []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostConsolePort}},
			},
		}, nil
}

func (d *MinIODriver) prepareConfig(confDir string) error {
	// MinIO does not need configuration files by default, but we ensure the directories exist
	dataDir := filepath.Join(d.DataDir, "data")
	return fileutil.EnsureDir(dataDir, 0755)
}

// Upgrade upgrades the MinIO service to target version
func (d *MinIODriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}

// GetCredentials returns credentials for MinIO
func (d *MinIODriver) GetCredentials() []credutil.Credential {
	port := d.Config.GetString("minio.port")
	if port == "" {
		port = "9000"
	}
	consolePort := d.Config.GetString("minio.console_port")
	if consolePort == "" {
		consolePort = "9001"
	}
	return []credutil.Credential{
		{Label: "API Address", Value: fmt.Sprintf("localhost:%s", port)},
		{Label: "Console Address", Value: fmt.Sprintf("localhost:%s", consolePort)},
		{Label: "Username", Value: d.rootUser},
		{Label: "Password", Value: d.rootPassword},
	}
}
