package docker

import (
	"time"

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
	return d.installWithSpec(d.containerSpec)
}

func (d *PostgresDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("5432/tcp")
	return &container.Config{
			Image: d.Image,
			Env:   []string{"POSTGRES_PASSWORD=" + d.password},
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD-SHELL", "pg_isready -U postgres"},
				Interval:    10 * time.Second,
				Timeout:     5 * time.Second,
				StartPeriod: 15 * time.Second,
				Retries:     10,
			},
		}, &container.HostConfig{
			Binds: []string{d.DataDir + ":/var/lib/postgresql/data"},
			PortBindings: nat.PortMap{
				port: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "5432"}},
			},
		}, nil
}

func (d *PostgresDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}
