package postgres

import (
	"OpsVault/cmd/common"
	"OpsVault/internal/driver"

	"github.com/spf13/cobra"
)

func (c *commandSet) newRestartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart PostgreSQL",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver("")
			if err != nil {
				return err
			}
			return drv.Restart()
		},
	}
}
