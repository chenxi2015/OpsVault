package docker

import (
	"context"
	"fmt"

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
	if err := d.EnsureReady(context.Background()); err != nil {
		return err
	}
	if d.Client == nil {
		return fmt.Errorf("docker client is not available")
	}
	cfg, hostCfg, err := d.containerSpec()
	if err != nil {
		return err
	}
	_, err = d.Client.ContainerCreate(context.Background(), cfg, hostCfg, nil, nil, d.ContainerName)
	return err
}

func (d *MySQLDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("3306/tcp")
	return &container.Config{
			Image: d.Image,
			Env:   []string{"MYSQL_ROOT_PASSWORD=" + d.rootPassword},
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
