package tui

import (
	"OpsVault/internal/driver"
	"OpsVault/internal/driver/binary"
	dockdrv "OpsVault/internal/driver/docker"

	"github.com/docker/docker/client"
	"github.com/spf13/viper"
)

type RuntimeStatusProvider struct {
	config        *viper.Viper
	dockerFactory func() (*client.Client, error)
}

func NewRuntimeStatusProvider(cfg *viper.Viper, dockerFactory func() (*client.Client, error)) RuntimeStatusProvider {
	return RuntimeStatusProvider{config: cfg, dockerFactory: dockerFactory}
}

func (p RuntimeStatusProvider) Services() []ServiceRef {
	cli, _ := p.dockerFactory() // Ignore error to let Nginx remain operational

	return []ServiceRef{
		{Name: "nginx", Driver: binary.NewNginxDriver(p.config)},
		{Name: "mysql", Driver: dockdrv.NewMySQLDriver(dockdrv.WrapClient(cli), p.config, "")},
		{Name: "redis", Driver: dockdrv.NewRedisDriver(dockdrv.WrapClient(cli), p.config, "")},
		{Name: "rocketmq", Driver: dockdrv.NewRocketMQDriver(dockdrv.WrapClient(cli), p.config)},
		{Name: "rabbitmq", Driver: dockdrv.NewRabbitMQDriver(dockdrv.WrapClient(cli), p.config, "", "")},
		{Name: "postgres", Driver: dockdrv.NewPostgresDriver(dockdrv.WrapClient(cli), p.config, "")},
		{Name: "elk", Driver: dockdrv.NewELKDriver(dockdrv.WrapClient(cli), p.config)},
	}
}

func (p RuntimeStatusProvider) Statuses() ([]driver.ServiceStatus, error) {
	services := p.Services()
	results := make([]driver.ServiceStatus, 0, len(services))
	for _, service := range services {
		status, err := service.Driver.Status()
		if err != nil {
			return nil, err
		}
		results = append(results, *status)
	}
	return results, nil
}
