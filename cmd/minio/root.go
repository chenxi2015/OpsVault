package minio

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

// NewCommand creates and returns a new cobra.Command for MinIO service management
func NewCommand(cfg *viper.Viper, dockerFactory func() (*client.Client, error)) *cobra.Command {
	c := &commandSet{config: cfg, dockerFactory: dockerFactory}
	cmd := &cobra.Command{Use: "minio", Short: "Manage MinIO"}
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

func (c *commandSet) driver(rootPassword string) (*docker.MinIODriver, error) {
	cli, err := c.dockerFactory()
	if err != nil {
		return nil, err
	}
	return docker.NewMinIODriver(docker.WrapClient(cli), c.config, rootPassword), nil
}
