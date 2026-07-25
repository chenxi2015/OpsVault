package jenkins

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

	cmd := &cobra.Command{
		Use:   "jenkins",
		Short: "Manage Jenkins",
	}
	cmd.AddCommand(
		c.newInstallCommand(),
		common.NewStartCmd("Jenkins", getMode, getDriver),
		common.NewStopCmd("Jenkins", getMode, getDriver),
		common.NewRestartCmd("Jenkins", getMode, getDriver),
		common.NewUninstallCmd("Jenkins", getMode, getDriver),
		c.newUpgradeCommand(),
		common.NewStatusCmd("Jenkins", getMode, getDriver),
		c.newLogCommand(),
	)
	return cmd
}

func (c *commandSet) driver() (*docker.JenkinsDriver, error) {
	cli, err := c.dockerFactory()
	if err != nil {
		return nil, err
	}
	wrapped := docker.WrapClient(cli)
	return docker.NewJenkinsDriver(wrapped, c.config), nil
}
