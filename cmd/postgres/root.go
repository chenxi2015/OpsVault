package postgres

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

	cmd := &cobra.Command{Use: "postgres", Short: "Manage PostgreSQL"}
	cmd.AddCommand(
		c.newInstallCommand(),
		common.NewStartCmd("PostgreSQL", getMode, getDriver),
		common.NewStopCmd("PostgreSQL", getMode, getDriver),
		common.NewRestartCmd("PostgreSQL", getMode, getDriver),
		common.NewUninstallCmd("PostgreSQL", getMode, getDriver),
		c.newUpgradeCommand(),
		common.NewStatusCmd("PostgreSQL", getMode, getDriver),
		c.newLogCommand(),
	)
	return cmd
}

func (c *commandSet) driver(password string) (*docker.PostgresDriver, error) {
	cli, err := c.dockerFactory()
	if err != nil {
		return nil, err
	}
	return docker.NewPostgresDriver(docker.WrapClient(cli), c.config, password), nil
}
