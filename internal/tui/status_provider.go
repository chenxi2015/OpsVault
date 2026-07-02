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

func (p RuntimeStatusProvider) Statuses() ([]driver.ServiceStatus, error) {
	cli, err := p.dockerFactory()
	if err != nil {
		return nil, err
	}

	services := []driver.ServiceDriver{
		binary.NewNginxDriver(p.config),
		dockdrv.NewMySQLDriver(dockdrv.WrapClient(cli), p.config, ""),
		dockdrv.NewRedisDriver(dockdrv.WrapClient(cli), p.config, ""),
		dockdrv.NewRocketMQDriver(dockdrv.WrapClient(cli), p.config),
		dockdrv.NewRabbitMQDriver(dockdrv.WrapClient(cli), p.config, "", ""),
		dockdrv.NewPostgresDriver(dockdrv.WrapClient(cli), p.config, ""),
	}

	results := make([]driver.ServiceStatus, 0, len(services))
	for _, service := range services {
		status, err := service.Status()
		if err != nil {
			return nil, err
		}
		results = append(results, *status)
	}
	return results, nil
}
