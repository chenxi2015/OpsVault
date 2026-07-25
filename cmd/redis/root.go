package redis

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

	cmd := &cobra.Command{Use: "redis", Short: "Manage Redis"}
	cmd.AddCommand(
		c.newInstallCommand(),
		common.NewStartCmd("Redis", getMode, getDriver),
		common.NewStopCmd("Redis", getMode, getDriver),
		common.NewRestartCmd("Redis", getMode, getDriver),
		common.NewUninstallCmd("Redis", getMode, getDriver),
		c.newUpgradeCommand(),
		common.NewStatusCmd("Redis", getMode, getDriver),
		c.newCliCommand(),
	)
	return cmd
}

func (c *commandSet) driver(password string) (*docker.RedisDriver, error) {
	cli, err := c.dockerFactory()
	if err != nil {
		return nil, err
	}
	return docker.NewRedisDriver(docker.WrapClient(cli), c.config, password), nil
}
