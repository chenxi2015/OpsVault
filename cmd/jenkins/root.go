package jenkins

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
	cmd := &cobra.Command{
		Use:   "jenkins",
		Short: "Manage Jenkins",
	}
	cmd.AddCommand(
		c.newInstallCommand(),
		c.newStartCommand(),
		c.newStopCommand(),
		c.newRestartCommand(),
		c.newUninstallCommand(),
		c.newUpgradeCommand(),
		c.newStatusCommand(),
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
