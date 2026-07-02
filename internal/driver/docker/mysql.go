package docker

import (
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/viper"
)

type MySQLDriver struct {
	*BaseDriver
	rootPassword string
}

func NewMySQLDriver(cli DockerClient, cfg *viper.Viper, rootPassword string) *MySQLDriver {
	base := NewBaseDriver("mysql", cli.Raw(), cfg, cfg.GetString("docker.images.mysql"), []string{"3306:3306"})
	return &MySQLDriver{BaseDriver: base, rootPassword: rootPassword}
}

func (d *MySQLDriver) Install() error {
	return d.installWithSpec(d.containerSpec)
}

func (d *MySQLDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("3306/tcp")
	return &container.Config{
			Image: d.Image,
			Env:   []string{"MYSQL_ROOT_PASSWORD=" + d.rootPassword},
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD-SHELL", "mysqladmin ping -h 127.0.0.1 -p$MYSQL_ROOT_PASSWORD || exit 1"},
				Interval:    10 * time.Second,
				Timeout:     5 * time.Second,
				StartPeriod: 20 * time.Second,
				Retries:     12,
			},
		}, &container.HostConfig{
			Binds: []string{d.DataDir + ":/var/lib/mysql"},
			PortBindings: nat.PortMap{
				port: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "3306"}},
			},
		}, nil
}

func (d *MySQLDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}
