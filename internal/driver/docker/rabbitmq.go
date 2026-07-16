package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"OpsVault/pkg/credutil"
	"OpsVault/pkg/rabbitmqconf"

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
	base := NewBaseDriver("rabbitmq", cli.Raw(), cfg, image, []string{fmt.Sprintf("%d:5672", port), fmt.Sprintf("%d:15672", uiPort)})
	drv := &RabbitMQDriver{BaseDriver: base, user: user, pass: pass}
	drv.PrepareConfig = drv.prepareConfig
	return drv
}

func (d *RabbitMQDriver) Install() error {
	if d.pass == "" {
		pwd := credutil.GenPassword(20)
		d.pass = pwd
		d.Config.Set("rabbitmq.admin_pwd", pwd)
		_ = d.Config.WriteConfig()
	}
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
				portAMQP: []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostPort}},
				portUI:   []nat.PortBinding{{HostIP: d.BindIP, HostPort: hostUIPort}},
			},
		}, nil
}

func (d *RabbitMQDriver) prepareConfig(confDir string) error {
	filePath := filepath.Join(confDir, "rabbitmq.conf")
	if _, err := os.Stat(filePath); err == nil {
		return nil
	}
	return os.WriteFile(filePath, []byte(rabbitmqconf.RenderRabbitMQConf(d.user, d.pass)), 0o644)
}

func (d *RabbitMQDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}

func (d *RabbitMQDriver) GetCredentials() []credutil.Credential {
	uiPort := d.Config.GetString("rabbitmq.ui_port")
	if uiPort == "" {
		uiPort = "15672"
	}
	amqpPort := d.Config.GetString("rabbitmq.port")
	if amqpPort == "" {
		amqpPort = "5672"
	}
	return []credutil.Credential{
		{Label: "管理界面", Value: fmt.Sprintf("http://localhost:%s", uiPort)},
		{Label: "AMQP 端口", Value: fmt.Sprintf("localhost:%s", amqpPort)},
		{Label: "用户名", Value: d.user},
		{Label: "密  码", Value: d.pass},
	}
}

