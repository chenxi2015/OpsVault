package nginx

import (
	"OpsVault/internal/driver/binary"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type commandSet struct {
	config *viper.Viper
}

func NewCommand(cfg *viper.Viper) *cobra.Command {
	c := &commandSet{config: cfg}
	cmd := &cobra.Command{
		Use:   "nginx",
		Short: "Manage Nginx with the binary driver",
	}
	cmd.AddCommand(
		c.newInstallCommand(),
		c.newStartCommand(),
		c.newStopCommand(),
		c.newRestartCommand(),
		c.newUninstallCommand(),
		c.newUpgradeCommand(),
		c.newVHostCommand(),
		c.newSSLCommand(),
		c.newStatusCommand(),
		c.newLogCommand(),
	)
	return cmd
}

func (c *commandSet) driver() *binary.NginxDriver {
	return binary.NewNginxDriver(c.config)
}
