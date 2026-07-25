package docker

import (
	"fmt"

	"OpsVault/pkg/credutil"

	"github.com/docker/docker/api/types/container"
	"github.com/spf13/viper"
)

// NodeExporterDriver represents the Docker driver for Node Exporter service
type NodeExporterDriver struct {
	*BaseDriver
}

// NewNodeExporterDriver creates and returns a new NodeExporterDriver instance
func NewNodeExporterDriver(cli DockerClient, cfg *viper.Viper) *NodeExporterDriver {
	port := cfg.GetInt("node_exporter.port")
	if port == 0 {
		port = 9100
	}
	image := cfg.GetString("node_exporter.image")
	if image == "" {
		image = "prom/node-exporter:latest"
	}

	base := NewBaseDriver("node-exporter", cli.Raw(), cfg, image, []string{
		fmt.Sprintf("%d:9100", port),
	})
	return &NodeExporterDriver{BaseDriver: base}
}

// Install runs the installer for Node Exporter container
func (d *NodeExporterDriver) Install() error {
	return d.installWithSpec(d.containerSpec)
}

func (d *NodeExporterDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	return &container.Config{
			Image: d.Image,
			Cmd: []string{
				"--path.procfs=/host/proc",
				"--path.sysfs=/host/sys",
				"--path.rootfs=/rootfs",
			},
		}, &container.HostConfig{
			NetworkMode: "host",
			PidMode:     "host",
			Binds: []string{
				toDockerBind("/proc", "/host/proc:ro"),
				toDockerBind("/sys", "/host/sys:ro"),
				toDockerBind("/", "/rootfs:ro"),
			},
		}, nil
}

// Upgrade upgrades the Node Exporter service to target version
func (d *NodeExporterDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}

// GetCredentials returns connection credentials for Node Exporter
func (d *NodeExporterDriver) GetCredentials() []credutil.Credential {
	port := d.Config.GetString("node_exporter.port")
	if port == "" {
		port = "9100"
	}
	return []credutil.Credential{
		{Label: "Metrics Endpoint", Value: fmt.Sprintf("http://localhost:%s/metrics", port)},
	}
}
