package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/viper"
)

type RedisDriver struct {
	*BaseDriver
	password string
}

func NewRedisDriver(cli DockerClient, cfg *viper.Viper, password string) *RedisDriver {
	base := NewBaseDriver("redis", cli.Raw(), cfg, cfg.GetString("docker.images.redis"), []string{"6379:6379"})
	return &RedisDriver{BaseDriver: base, password: password}
}

func (d *RedisDriver) Install() error {
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

func (d *RedisDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("6379/tcp")
	cmd := []string{"redis-server", "--appendonly", "yes"}
	if d.password != "" {
		cmd = append(cmd, "--requirepass", d.password)
	}
	return &container.Config{
			Image: d.Image,
			Cmd:   cmd,
		}, &container.HostConfig{
			Binds: []string{d.DataDir + ":/data"},
			PortBindings: nat.PortMap{
				port: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "6379"}},
			},
		}, nil
}

func (d *RedisDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}
