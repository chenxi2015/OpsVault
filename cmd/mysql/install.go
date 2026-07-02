package mysql

import (
	"OpsVault/cmd/common"
	"OpsVault/internal/driver"

	"github.com/spf13/cobra"
)

func (c *commandSet) newInstallCommand() *cobra.Command {
	var rootPwd string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install MySQL in Docker mode",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver(rootPwd)
			if err != nil {
				return err
			}
			return drv.Install()
		},
	}
	cmd.Flags().StringVar(&rootPwd, "root-pwd", "root", "MySQL root password")
	return cmd
}
