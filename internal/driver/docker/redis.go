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
	port := nat.Port("6379/tcp")
	cmd := []string{"redis-server", "--appendonly", "yes"}
	if d.password != "" {
		cmd = append(cmd, "--requirepass", d.password)
	}
	_, err := d.Client.ContainerCreate(context.Background(), &container.Config{
		Image: d.Image,
		Cmd:   cmd,
	}, &container.HostConfig{
		Binds: []string{d.DataDir + ":/data"},
		PortBindings: nat.PortMap{
			port: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "6379"}},
		},
	}, nil, nil, d.ContainerName)
	return err
}
