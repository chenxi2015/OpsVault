package docker

import (
	"time"

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
	return d.installWithSpec(d.containerSpec)
}

func (d *RedisDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("6379/tcp")
	cmd := []string{"redis-server", "--appendonly", "yes"}
	if d.password != "" {
		cmd = append(cmd, "--requirepass", d.password)
	}
	healthCmd := `redis-cli ping | grep PONG`
	if d.password != "" {
		healthCmd = `redis-cli -a "$REDIS_PASSWORD" ping | grep PONG`
	}
	return &container.Config{
			Image: d.Image,
			Cmd:   cmd,
			Env:   []string{"REDIS_PASSWORD=" + d.password},
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD-SHELL", healthCmd},
				Interval:    10 * time.Second,
				Timeout:     5 * time.Second,
				StartPeriod: 10 * time.Second,
				Retries:     10,
			},
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
