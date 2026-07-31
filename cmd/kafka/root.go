package kafka

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
		return c.driver()
	}

	cmd := &cobra.Command{Use: "kafka", Short: "Manage Kafka"}
	cmd.AddCommand(
		c.newInstallCommand(),
		common.NewStartCmd("Kafka", getMode, getDriver),
		common.NewStopCmd("Kafka", getMode, getDriver),
		common.NewRestartCmd("Kafka", getMode, getDriver),
		common.NewUninstallCmd("Kafka", getMode, getDriver),
		c.newUpgradeCommand(),
		common.NewStatusCmd("Kafka", getMode, getDriver),
		c.newLogCommand(),
		c.newVersionCommand(),
		c.newShCommand(),
	)
	return cmd
}

func (c *commandSet) driver() (*docker.KafkaDriver, error) {
	cli, err := c.dockerFactory()
	if err != nil {
		return nil, err
	}
	return docker.NewKafkaDriver(docker.WrapClient(cli), c.config), nil
}
