package nacos

import (
	"OpsVault/cmd/common"
	"OpsVault/internal/driver"
	"OpsVault/pkg/credutil"

	"github.com/spf13/cobra"
)

func (c *commandSet) newInstallCommand() *cobra.Command {
	var (
		authToken string
	)
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Nacos",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			if authToken == "" {
				authToken = c.config.GetString("nacos.auth_token")
			}
			drv, err := c.driver(authToken)
			if err != nil {
				return err
			}
			if err := drv.Install(); err != nil {
				return err
			}
			credutil.PrintCredentials("Nacos", drv.GetCredentials())
			return nil
		},
	}
	cmd.Flags().StringVar(&authToken, "token", "", "Nacos JWT authentication token secret (min 32 chars base64)")
	return cmd
}
