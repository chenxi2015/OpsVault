package qdrant

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

// NewCommand creates and returns a new cobra.Command for Qdrant service management
func NewCommand(cfg *viper.Viper, dockerFactory func() (*client.Client, error)) *cobra.Command {
	c := &commandSet{config: cfg, dockerFactory: dockerFactory}
	getMode := func() string { return cfg.GetString("mode") }
	getDriver := func() (driver.ServiceDriver, error) {
		return c.driver("")
	}

	cmd := &cobra.Command{Use: "qdrant", Short: "Manage Qdrant vector database"}
	cmd.AddCommand(
		c.newInstallCommand(),
		common.NewStartCmd("Qdrant", getMode, getDriver),
		common.NewStopCmd("Qdrant", getMode, getDriver),
		common.NewRestartCmd("Qdrant", getMode, getDriver),
		common.NewUninstallCmd("Qdrant", getMode, getDriver),
		c.newUpgradeCommand(),
		common.NewStatusCmd("Qdrant", getMode, getDriver),
		c.newLogCommand(),
	)
	return cmd
}

func (c *commandSet) driver(apiKey string) (*docker.QdrantDriver, error) {
	cli, err := c.dockerFactory()
	if err != nil {
		return nil, err
	}
	return docker.NewQdrantDriver(docker.WrapClient(cli), c.config, apiKey), nil
}
