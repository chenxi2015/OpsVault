package rabbitmq

import (
	"fmt"

	"OpsVault/cmd/common"
	"OpsVault/internal/driver"
	"OpsVault/pkg/credutil"

	"github.com/spf13/cobra"
)

func (c *commandSet) newInstallCommand() *cobra.Command {
	var (
		user      string
		pass      string
		randomPwd bool
	)
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install RabbitMQ",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			if randomPwd {
				pass = credutil.GenPassword(20)
			} else if pass == "" {
				pass = c.config.GetString("rabbitmq.admin_pwd")
			}
			if user == "" {
				user = c.config.GetString("rabbitmq.admin_user")
			}
			if user == "" {
				user = "admin"
			}
			if pass == "" {
				return fmt.Errorf("RabbitMQ admin password is required: use --admin-pwd <pwd> or --random-pwd")
			}
			drv, err := c.driver(user, pass)
			if err != nil {
				return err
			}
			if err := drv.Install(); err != nil {
				return err
			}
			credutil.PrintCredentials("RabbitMQ", drv.GetCredentials())
			return nil
		},
	}
	cmd.Flags().StringVar(&user, "admin-user", "", "RabbitMQ admin user (default: admin)")
	cmd.Flags().StringVar(&pass, "admin-pwd", "", "RabbitMQ admin password")
	cmd.Flags().BoolVar(&randomPwd, "random-pwd", false, "Generate a secure random password")
	return cmd
}

