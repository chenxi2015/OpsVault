package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/viper"
)

type PostgresDriver struct {
	*BaseDriver
	password string
}

func NewPostgresDriver(cli DockerClient, cfg *viper.Viper, password string) *PostgresDriver {
	base := NewBaseDriver("postgres", cli.Raw(), cfg, cfg.GetString("docker.images.postgres"), []string{"5432:5432"})
	return &PostgresDriver{BaseDriver: base, password: password}
}

func (d *PostgresDriver) Install() error {
	if err := d.EnsureReady(context.Background()); err != nil {
		return err
	}
	if d.Client == nil {
		return fmt.Errorf("docker client is not available")
	}
	port := nat.Port("5432/tcp")
	_, err := d.Client.ContainerCreate(context.Background(), &container.Config{
		Image: d.Image,
		Env:   []string{"POSTGRES_PASSWORD=" + d.password},
	}, &container.HostConfig{
		Binds: []string{d.DataDir + ":/var/lib/postgresql/data"},
		PortBindings: nat.PortMap{
			port: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "5432"}},
		},
	}, nil, nil, d.ContainerName)
	return err
}
