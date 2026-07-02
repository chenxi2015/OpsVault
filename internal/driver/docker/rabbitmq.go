package docker

import (
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
	base := NewBaseDriver("rabbitmq", cli.Raw(), cfg, cfg.GetString("docker.images.rabbitmq"), []string{"5672:5672", "15672:15672"})
	return &RabbitMQDriver{BaseDriver: base, user: user, pass: pass}
}

func (d *RabbitMQDriver) Install() error {
	return d.installWithSpec(d.containerSpec)
}

func (d *RabbitMQDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	portAMQP := nat.Port("5672/tcp")
	portUI := nat.Port("15672/tcp")
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
			Binds: []string{d.DataDir + ":/var/lib/rabbitmq"},
			PortBindings: nat.PortMap{
				portAMQP: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "5672"}},
				portUI:   []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "15672"}},
			},
		}, nil
}

func (d *RabbitMQDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}
