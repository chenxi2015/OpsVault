package grafana

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

// NewCommand creates and returns a new cobra.Command for Grafana service management
func NewCommand(cfg *viper.Viper, dockerFactory func() (*client.Client, error)) *cobra.Command {
	c := &commandSet{config: cfg, dockerFactory: dockerFactory}
	getMode := func() string { return cfg.GetString("mode") }
	getDriver := func() (driver.ServiceDriver, error) {
		return c.driver("")
	}

	cmd := &cobra.Command{Use: "grafana", Short: "Manage Grafana dashboard server"}
	cmd.AddCommand(
		c.newInstallCommand(),
		common.NewStartCmd("Grafana", getMode, getDriver),
		common.NewStopCmd("Grafana", getMode, getDriver),
		common.NewRestartCmd("Grafana", getMode, getDriver),
		common.NewUninstallCmd("Grafana", getMode, getDriver),
		c.newUpgradeCommand(),
		common.NewStatusCmd("Grafana", getMode, getDriver),
		c.newLogCommand(),
	)
	return cmd
}

func (c *commandSet) driver(adminPassword string) (*docker.GrafanaDriver, error) {
	cli, err := c.dockerFactory()
	if err != nil {
		return nil, err
	}
	return docker.NewGrafanaDriver(docker.WrapClient(cli), c.config, adminPassword), nil
}
