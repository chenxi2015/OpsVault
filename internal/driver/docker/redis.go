package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"OpsVault/pkg/credutil"

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
		_ = d.Config.WriteConfig()
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
			Binds: []string{
				filepath.Join(d.DataDir, "data") + ":/data",
				filepath.Join(d.DataDir, "conf", "redis.conf") + ":/usr/local/etc/redis/redis.conf",
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
	content := `bind 0.0.0.0
protected-mode no
port 6379
tcp-backlog 511
timeout 0
tcp-keepalive 300
daemonize no
supervised no
loglevel notice
databases 16
always-show-logo yes
save 900 1
save 300 10
save 60 10000
stop-writes-on-bgsave-error yes
rdbcompression yes
rdbchecksum yes
dbfilename dump.rdb
dir /data
appendonly yes
appendfilename "appendonly.aof"
appendfsync everysec
no-appendfsync-on-rewrite no
auto-aof-rewrite-percentage 100
auto-aof-rewrite-min-size 64mb
aof-load-truncated yes
aof-use-rdb-preamble yes
`
	pwd := d.password
	if pwd == "" {
		pwd = d.Config.GetString("redis.password")
	}
	if pwd != "" {
		content += fmt.Sprintf("\nrequirepass %s\n", pwd)
	}

	return os.WriteFile(filePath, []byte(content), 0o644)
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

