package docker

import (
	"fmt"
	"path/filepath"
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
	port := cfg.GetInt("postgres.port")
	if port == 0 {
		port = 5432
	}
	image := cfg.GetString("postgres.image")
	if image == "" {
		image = "postgres:15"
	}
	if password == "" {
		password = cfg.GetString("postgres.password")
	}
	base := NewBaseDriver("postgres", cli.Raw(), cfg, image, []string{fmt.Sprintf("%d:%d", port, port)})
	return &PostgresDriver{BaseDriver: base, password: password}
}

func (d *PostgresDriver) Install() error {
	return d.installWithSpec(d.containerSpec)
}

func (d *PostgresDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("5432/tcp")
	hostPort := d.Config.GetString("postgres.port")
	if hostPort == "" {
		hostPort = "5432"
	}
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
			Binds: []string{filepath.Join(d.DataDir, "data") + ":/var/lib/postgresql/data"},
			PortBindings: nat.PortMap{
				port: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: hostPort}},
			},
		}, nil
}

func (d *PostgresDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}
