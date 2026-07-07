package redis

import (
	"OpsVault/internal/driver/docker"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type commandSet struct {
	config        *viper.Viper
	dockerFactory func() (*client.Client, error)
}

func NewCommand(cfg *viper.Viper, dockerFactory func() (*client.Client, error)) *cobra.Command {
	c := &commandSet{config: cfg, dockerFactory: dockerFactory}
	cmd := &cobra.Command{Use: "redis", Short: "Manage Redis"}
	cmd.AddCommand(
		c.newInstallCommand(),
		c.newStartCommand(),
		c.newStopCommand(),
		c.newRestartCommand(),
		c.newUninstallCommand(),
		c.newUpgradeCommand(),
		c.newStatusCommand(),
		c.newCliCommand(),
	)
	return cmd
}

func (c *commandSet) driver(password string) (*docker.RedisDriver, error) {
	cli, err := c.dockerFactory()
	if err != nil {
		return nil, err
	}
	return docker.NewRedisDriver(docker.WrapClient(cli), c.config, password), nil
}
