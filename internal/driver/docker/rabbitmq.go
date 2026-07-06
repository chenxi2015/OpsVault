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

type RabbitMQDriver struct {
	*BaseDriver
	user string
	pass string
}

func NewRabbitMQDriver(cli DockerClient, cfg *viper.Viper, user, pass string) *RabbitMQDriver {
	port := cfg.GetInt("rabbitmq.port")
	if port == 0 {
		port = 5672
	}
	uiPort := cfg.GetInt("rabbitmq.ui_port")
	if uiPort == 0 {
		uiPort = 15672
	}
	image := cfg.GetString("rabbitmq.image")
	if image == "" {
		image = "rabbitmq:3-management"
	}
	if user == "" {
		user = cfg.GetString("rabbitmq.admin_user")
	}
	if user == "" {
		user = "admin"
	}
	if pass == "" {
		pass = cfg.GetString("rabbitmq.admin_pwd")
	}
	if pass == "" {
		pass = "password"
	}
	base := NewBaseDriver("rabbitmq", cli.Raw(), cfg, image, []string{fmt.Sprintf("%d:%d", port, port), fmt.Sprintf("%d:%d", uiPort, uiPort)})
	drv := &RabbitMQDriver{BaseDriver: base, user: user, pass: pass}
	drv.PrepareConfig = drv.prepareConfig
	return drv
}

func (d *RabbitMQDriver) Install() error {
	return d.installWithSpec(d.containerSpec)
}

func (d *RabbitMQDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	portAMQP := nat.Port("5672/tcp")
	portUI := nat.Port("15672/tcp")
	hostPort := d.Config.GetString("rabbitmq.port")
	if hostPort == "" {
		hostPort = "5672"
	}
	hostUIPort := d.Config.GetString("rabbitmq.ui_port")
	if hostUIPort == "" {
		hostUIPort = "15672"
	}

	return &container.Config{
			Image: d.Image,
			Env: []string{
				"RABBITMQ_DEFAULT_USER=" + d.user,
				"RABBITMQ_DEFAULT_PASS=" + d.pass,
			},
			Healthcheck: &container.HealthConfig{
				Test:        []string{"CMD-SHELL", "rabbitmq-diagnostics -q ping"},
				Interval:    10 * time.Second,
				Timeout:     5 * time.Second,
				StartPeriod: 20 * time.Second,
				Retries:     12,
			},
		}, &container.HostConfig{
			Binds: []string{
				filepath.Join(d.DataDir, "data") + ":/var/lib/rabbitmq",
				filepath.Join(d.DataDir, "conf", "rabbitmq.conf") + ":/etc/rabbitmq/rabbitmq.conf",
			},
			PortBindings: nat.PortMap{
				portAMQP: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: hostPort}},
				portUI:   []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: hostUIPort}},
			},
		}, nil
}

func (d *RabbitMQDriver) prepareConfig(confDir string) error {
	filePath := filepath.Join(confDir, "rabbitmq.conf")
	if _, err := os.Stat(filePath); err == nil {
		return nil
	}
	content := fmt.Sprintf(`loopback_users.guest = false
listeners.tcp.default = 5672
default_user = %s
default_pass = %s
`, d.user, d.pass)
	return os.WriteFile(filePath, []byte(content), 0o644)
}

func (d *RabbitMQDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}
