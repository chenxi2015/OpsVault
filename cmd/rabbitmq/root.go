package rabbitmq

import (
	"OpsVault/cmd/common"
	"OpsVault/internal/driver"
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
	getMode := func() string { return cfg.GetString("mode") }
	getDriver := func() (driver.ServiceDriver, error) {
		return c.driver("", "")
	}

	cmd := &cobra.Command{Use: "rabbitmq", Short: "Manage RabbitMQ"}
	cmd.AddCommand(
		c.newInstallCommand(),
		common.NewStartCmd("RabbitMQ", getMode, getDriver),
		common.NewStopCmd("RabbitMQ", getMode, getDriver),
		common.NewRestartCmd("RabbitMQ", getMode, getDriver),
		common.NewUninstallCmd("RabbitMQ", getMode, getDriver),
		c.newUpgradeCommand(),
		common.NewStatusCmd("RabbitMQ", getMode, getDriver),
	)
	return cmd
}

func (c *commandSet) driver(user, pass string) (*docker.RabbitMQDriver, error) {
	cli, err := c.dockerFactory()
	if err != nil {
		return nil, err
	}
	return docker.NewRabbitMQDriver(docker.WrapClient(cli), c.config, user, pass), nil
}
