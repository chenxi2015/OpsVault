package rabbitmq

import (
	"OpsVault/cmd/common"
	"OpsVault/internal/driver"

	"github.com/spf13/cobra"
)

func (c *commandSet) newInstallCommand() *cobra.Command {
	var user string
	var pass string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install RabbitMQ",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver(user, pass)
			if err != nil {
				return err
			}
			return drv.Install()
		},
	}
	cmd.Flags().StringVar(&user, "admin-user", "admin", "RabbitMQ admin user")
	cmd.Flags().StringVar(&pass, "admin-pwd", "123456", "RabbitMQ admin password")
	return cmd
}
