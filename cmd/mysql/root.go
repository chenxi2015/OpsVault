package mysql

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
		return c.driver("")
	}

	cmd := &cobra.Command{
		Use:   "mysql",
		Short: "Manage MySQL",
	}
	cmd.AddCommand(
		c.newInstallCommand(),
		common.NewStartCmd("MySQL", getMode, getDriver),
		common.NewStopCmd("MySQL", getMode, getDriver),
		common.NewRestartCmd("MySQL", getMode, getDriver),
		common.NewUninstallCmd("MySQL", getMode, getDriver),
		c.newUpgradeCommand(),
		common.NewStatusCmd("MySQL", getMode, getDriver),
		c.newLogCommand(),
		c.newExecCommand(),
	)
	return cmd
}

func (c *commandSet) driver(rootPassword string) (*docker.MySQLDriver, error) {
	cli, err := c.dockerFactory()
	if err != nil {
		return nil, err
	}
	wrapped := docker.WrapClient(cli)
	return docker.NewMySQLDriver(wrapped, c.config, rootPassword), nil
}
