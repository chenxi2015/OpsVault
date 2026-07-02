package postgres

import (
	"OpsVault/cmd/common"
	"OpsVault/internal/driver"

	"github.com/spf13/cobra"
)

func (c *commandSet) newInstallCommand() *cobra.Command {
	var password string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install PostgreSQL",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver(password)
			if err != nil {
				return err
			}
			return drv.Install()
		},
	}
	cmd.Flags().StringVar(&password, "pwd", "postgres", "PostgreSQL password")
	return cmd
}
