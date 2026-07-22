package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"OpsVault/pkg/credutil"
	"OpsVault/pkg/fileutil"
	"OpsVault/pkg/redisconf"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/viper"
)

type RedisDriver struct {
	*BaseDriver
	password string
}

func NewRedisDriver(cli DockerClient, cfg *viper.Viper, password string) *RedisDriver {
	port := cfg.GetInt("redis.port")
	if port == 0 {
		port = 6379
	}
	image := cfg.GetString("redis.image")
	if image == "" {
		image = "redis:7-alpine"
	}
	if password == "" {
		password = cfg.GetString("redis.password")
	}
	base := NewBaseDriver("redis", cli.Raw(), cfg, image, []string{fmt.Sprintf("%d:6379", port)})
	drv := &RedisDriver{BaseDriver: base, password: password}
	drv.PrepareConfig = drv.prepareConfig
	return drv
}

func (d *RedisDriver) Install() error {
	if d.password == "" {
		pwd := credutil.GenPassword(20)
		d.password = pwd
		d.Config.Set("redis.password", pwd)
		cfgPath := d.Config.ConfigFileUsed()
		if cfgPath == "" {
			cfgPath = fileutil.GetDefaultWriteConfigPath()
		}
		_ = fileutil.UpdateYAMLValue(cfgPath, "redis", "password", pwd)
	}
	return d.installWithSpec(d.containerSpec)
}

func (d *RedisDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("6379/tcp")
	hostPort := d.Config.GetString("redis.port")
	if hostPort == "" {
		hostPort = "6379"
	}

	cmd := []string{"redis-server", "/usr/local/etc/redis/redis.conf"}

	var env []string
	if d.password != "" {
		env = []string{"REDISCLI_AUTH=" + d.password}
	}

	healthCmd := `redis-cli ping | grep PONG`
	return &container.Config{
			Image: d.Image,
			Cmd:   cmd,
			Env:   env,
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD-SHELL", healthCmd},
				Interval:    10 * time.Second,
				Timeout:     5 * time.Second,
				StartPeriod: 10 * time.Second,
				Retries:     10,
			},
		}, &container.HostConfig{
			Binds: []string{
				toDockerBind(filepath.Join(d.DataDir, "data"), "/data"),
				toDockerBind(filepath.Join(d.DataDir, "conf", "redis.conf"), "/usr/local/etc/redis/redis.conf"),
			},
			PortBindings: nat.PortMap{
				port: []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostPort}},
			},
		}, nil
}

func (d *RedisDriver) prepareConfig(confDir string) error {
	filePath := filepath.Join(confDir, "redis.conf")
	if _, err := os.Stat(filePath); err == nil {
		return nil
	}
	pwd := d.password
	if pwd == "" {
		pwd = d.Config.GetString("redis.password")
	}
	return os.WriteFile(filePath, []byte(redisconf.RenderRedisCnf(pwd)), 0o644)
}

func (d *RedisDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}

func (d *RedisDriver) GetCredentials() []credutil.Credential {
	port := d.Config.GetString("redis.port")
	if port == "" {
		port = "6379"
	}
	pwd := d.password
	if pwd == "" {
		pwd = "(无认证)"
	}
	return []credutil.Credential{
		{Label: "主机", Value: fmt.Sprintf("localhost:%s", port)},
		{Label: "密  码", Value: pwd},
	}
}
