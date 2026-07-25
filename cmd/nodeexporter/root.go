package nodeexporter

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

// NewCommand creates and returns a new cobra.Command for Node Exporter service management
func NewCommand(cfg *viper.Viper, dockerFactory func() (*client.Client, error)) *cobra.Command {
	c := &commandSet{config: cfg, dockerFactory: dockerFactory}
	getMode := func() string { return cfg.GetString("mode") }
	getDriver := func() (driver.ServiceDriver, error) {
		return c.driver()
	}

	cmd := &cobra.Command{Use: "node-exporter", Short: "Manage Node Exporter hardware & system metrics agent"}
	cmd.AddCommand(
		c.newInstallCommand(),
		common.NewStartCmd("Node Exporter", getMode, getDriver),
		common.NewStopCmd("Node Exporter", getMode, getDriver),
		common.NewRestartCmd("Node Exporter", getMode, getDriver),
		common.NewUninstallCmd("Node Exporter", getMode, getDriver),
		c.newUpgradeCommand(),
		common.NewStatusCmd("Node Exporter", getMode, getDriver),
		c.newLogCommand(),
	)
	return cmd
}

func (c *commandSet) driver() (*docker.NodeExporterDriver, error) {
	cli, err := c.dockerFactory()
	if err != nil {
		return nil, err
	}
	return docker.NewNodeExporterDriver(docker.WrapClient(cli), c.config), nil
}
