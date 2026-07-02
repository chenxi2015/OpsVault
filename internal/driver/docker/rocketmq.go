package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/viper"
)

type RocketMQDriver struct {
	*BaseDriver
}

func NewRocketMQDriver(cli DockerClient, cfg *viper.Viper) *RocketMQDriver {
	base := NewBaseDriver("rocketmq", cli.Raw(), cfg, cfg.GetString("docker.images.rocketmq"), []string{"9876:9876"})
	return &RocketMQDriver{BaseDriver: base}
}

func (d *RocketMQDriver) Install() error {
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

func (d *RocketMQDriver) containerSpec() (*container.Config, *container.HostConfig, error) {
	port := nat.Port("9876/tcp")
	return &container.Config{
			Image: d.Image,
			Env:   []string{"NAMESRV_ADDR=127.0.0.1:9876"},
			Cmd:   []string{"sh", "mqbroker", "-n", "127.0.0.1:9876", "autoCreateTopicEnable=true"},
		}, &container.HostConfig{
			Binds: []string{d.DataDir + ":/home/rocketmq/store"},
			PortBindings: nat.PortMap{
				port: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "9876"}},
			},
		}, nil
}

func (d *RocketMQDriver) Upgrade(targetVersion string) error {
	return d.recreateWithImage(targetVersion, d.containerSpec)
}

func (d *RocketMQDriver) Version() string {
	return d.Image
}

func (d *RocketMQDriver) DLQStat() map[string]string {
	return map[string]string{
		"status": "not implemented",
	}
}
