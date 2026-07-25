package prometheus

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

// NewCommand creates and returns a new cobra.Command for Prometheus service management
func NewCommand(cfg *viper.Viper, dockerFactory func() (*client.Client, error)) *cobra.Command {
	c := &commandSet{config: cfg, dockerFactory: dockerFactory}
	getMode := func() string { return cfg.GetString("mode") }
	getDriver := func() (driver.ServiceDriver, error) {
		return c.driver()
	}

	cmd := &cobra.Command{Use: "prometheus", Short: "Manage Prometheus monitoring server"}
	cmd.AddCommand(
		c.newInstallCommand(),
		common.NewStartCmd("Prometheus", getMode, getDriver),
		common.NewStopCmd("Prometheus", getMode, getDriver),
		common.NewRestartCmd("Prometheus", getMode, getDriver),
		common.NewUninstallCmd("Prometheus", getMode, getDriver),
		c.newUpgradeCommand(),
		common.NewStatusCmd("Prometheus", getMode, getDriver),
		c.newLogCommand(),
		c.newReloadCommand(),
	)
	return cmd
}

func (c *commandSet) driver() (*docker.PrometheusDriver, error) {
	cli, err := c.dockerFactory()
	if err != nil {
		return nil, err
	}
	return docker.NewPrometheusDriver(docker.WrapClient(cli), c.config), nil
}
