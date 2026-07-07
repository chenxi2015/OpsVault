package rocketmq

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
	cmd := &cobra.Command{Use: "rocketmq", Short: "Manage RocketMQ"}
	cmd.AddCommand(
		c.newInstallCommand(),
		c.newStartCommand(),
		c.newStopCommand(),
		c.newRestartCommand(),
		c.newUninstallCommand(),
		c.newUpgradeCommand(),
		c.newVersionCommand(),
		c.newDLQCommand(),
		c.newLogCommand(),
		c.newShCommand(),
	)
	return cmd
}

func (c *commandSet) driver() (*docker.RocketMQDriver, error) {
	cli, err := c.dockerFactory()
	if err != nil {
		return nil, err
	}
	return docker.NewRocketMQDriver(docker.WrapClient(cli), c.config), nil
}
