package elk

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

	cmd := &cobra.Command{Use: "elk", Short: "Manage ELK (Elasticsearch, Logstash, Kibana) Stack"}
	cmd.AddCommand(
		c.newInstallCommand(),
		common.NewStartCmd("ELK Stack", getMode, getDriver),
		common.NewStopCmd("ELK Stack", getMode, getDriver),
		common.NewRestartCmd("ELK Stack", getMode, getDriver),
		common.NewUninstallCmd("ELK Stack", getMode, getDriver),
		c.newUpgradeCommand(),
		common.NewStatusCmd("ELK Stack", getMode, getDriver),
		c.newLogCommand(),
	)
	return cmd
}

func (c *commandSet) driver() (*docker.ELKDriver, error) {
	cli, err := c.dockerFactory()
	if err != nil {
		return nil, err
	}
	return docker.NewELKDriver(docker.WrapClient(cli), c.config), nil
}
