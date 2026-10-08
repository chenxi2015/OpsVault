package milvus

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

// NewCommand creates and returns a new cobra.Command for Milvus service management
func NewCommand(cfg *viper.Viper, dockerFactory func() (*client.Client, error)) *cobra.Command {
	c := &commandSet{config: cfg, dockerFactory: dockerFactory}
	getMode := func() string { return cfg.GetString("mode") }
	getDriver := func() (driver.ServiceDriver, error) {
		return c.driver()
	}

	cmd := &cobra.Command{Use: "milvus", Short: "Manage Milvus vector database"}
	cmd.AddCommand(
		c.newInstallCommand(),
		common.NewStartCmd("Milvus", getMode, getDriver),
		common.NewStopCmd("Milvus", getMode, getDriver),
		common.NewRestartCmd("Milvus", getMode, getDriver),
		common.NewUninstallCmd("Milvus", getMode, getDriver),
		c.newUpgradeCommand(),
		common.NewStatusCmd("Milvus", getMode, getDriver),
		c.newLogCommand(),
	)
	return cmd
}

func (c *commandSet) driver() (*docker.MilvusDriver, error) {
	cli, err := c.dockerFactory()
	if err != nil {
		return nil, err
	}
	return docker.NewMilvusDriver(docker.WrapClient(cli), c.config), nil
}

