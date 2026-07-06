package docker

import (
	"fmt"
	"os"
	"path/filepath"
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
	port := cfg.GetInt("mysql.port")
	if port == 0 {
		port = 3306
	}
	image := cfg.GetString("mysql.image")
	if image == "" {
		image = "mysql:8.0"
	}
	if rootPassword == "" {
		rootPassword = cfg.GetString("mysql.root_password")
	}
	if rootPassword == "" {
		rootPassword = "root"
	}
	base := NewBaseDriver("mysql", cli.Raw(), cfg, image, []string{fmt.Sprintf("%d:%d", port, port)})
	drv := &MySQLDriver{BaseDriver: base, rootPassword: rootPassword}
	drv.PrepareConfig = drv.prepareConfig
	return drv
}

func (d *MySQLDriver) Install() error {
	return d.installWithSpec(d.containerSpec)
}

func (d *MySQLDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("3306/tcp")
	hostPort := d.Config.GetString("mysql.port")
	if hostPort == "" {
		hostPort = "3306"
	}
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
			Binds: []string{
				filepath.Join(d.DataDir, "data") + ":/var/lib/mysql",
				filepath.Join(d.DataDir, "conf", "my.cnf") + ":/etc/mysql/conf.d/my.cnf",
			},
			PortBindings: nat.PortMap{
				port: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: hostPort}},
			},
		}, nil
}

func (d *MySQLDriver) prepareConfig(confDir string) error {
	filePath := filepath.Join(confDir, "my.cnf")
	if _, err := os.Stat(filePath); err == nil {
		return nil
	}
	content := `[mysqld]
user=mysql
default-storage-engine=InnoDB
character-set-server=utf8mb4
collation-server=utf8mb4_unicode_ci

[client]
default-character-set=utf8mb4
`
	return os.WriteFile(filePath, []byte(content), 0o644)
}

func (d *MySQLDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}
