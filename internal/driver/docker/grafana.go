package docker

import (
	"fmt"
	"path/filepath"

	"OpsVault/pkg/credutil"
	"OpsVault/pkg/fileutil"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/viper"
)

// GrafanaDriver represents the Docker driver for Grafana service
type GrafanaDriver struct {
	*BaseDriver
	adminUser     string
	adminPassword string
}

// NewGrafanaDriver creates and returns a new GrafanaDriver instance
func NewGrafanaDriver(cli DockerClient, cfg *viper.Viper, adminPassword string) *GrafanaDriver {
	port := cfg.GetInt("grafana.port")
	if port == 0 {
		port = 3000
	}
	image := cfg.GetString("grafana.image")
	if image == "" {
		image = "grafana/grafana:latest"
	}
	adminUser := cfg.GetString("grafana.admin_user")
	if adminUser == "" {
		adminUser = "admin"
	}
	if adminPassword == "" {
		adminPassword = cfg.GetString("grafana.admin_password")
	}

	base := NewBaseDriver("grafana", cli.Raw(), cfg, image, []string{
		fmt.Sprintf("%d:3000", port),
	})
	drv := &GrafanaDriver{BaseDriver: base, adminUser: adminUser, adminPassword: adminPassword}
	drv.PrepareConfig = drv.prepareConfig
	return drv
}

// Install runs the installer for Grafana container
func (d *GrafanaDriver) Install() error {
	if d.adminPassword == "" {
		pwd := credutil.GenPassword(20)
		d.adminPassword = pwd
		d.Config.Set("grafana.admin_password", pwd)
		cfgPath := d.Config.ConfigFileUsed()
		if cfgPath == "" {
			cfgPath = fileutil.GetDefaultWriteConfigPath()
		}
		_ = fileutil.UpdateYAMLValue(cfgPath, "grafana", "admin_password", pwd)
	}
	return d.installWithSpec(d.containerSpec)
}

func (d *GrafanaDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("3000/tcp")
	hostPort := d.Config.GetString("grafana.port")
	if hostPort == "" {
		hostPort = "3000"
	}
	dataPath := filepath.Join(d.DataDir, "data")

	return &container.Config{
			Image: d.Image,
			Env: []string{
				"GF_SECURITY_ADMIN_USER=" + d.adminUser,
				"GF_SECURITY_ADMIN_PASSWORD=" + d.adminPassword,
			},
		}, &container.HostConfig{
			Binds: []string{
				toDockerBind(dataPath, "/var/lib/grafana"),
			},
			PortBindings: nat.PortMap{
				port: []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostPort}},
			},
		}, nil
}

func (d *GrafanaDriver) prepareConfig(confDir string) error {
	dataPath := filepath.Join(d.DataDir, "data")
	return fileutil.EnsureDir(dataPath, 0755)
}

// Upgrade upgrades the Grafana service to target version
func (d *GrafanaDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}

// GetCredentials returns connection credentials for Grafana
func (d *GrafanaDriver) GetCredentials() []credutil.Credential {
	port := d.Config.GetString("grafana.port")
	if port == "" {
		port = "3000"
	}
	return []credutil.Credential{
		{Label: "Web UI", Value: fmt.Sprintf("http://localhost:%s", port)},
		{Label: "Username", Value: d.adminUser},
		{Label: "Password", Value: d.adminPassword},
	}
}
