package postgres

import (
	"OpsVault/cmd/common"
	"OpsVault/internal/driver"
	"fmt"

	"github.com/spf13/cobra"
)

func (c *commandSet) newLogCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "log",
		Short: "Show PostgreSQL logs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver("")
			if err != nil {
				return err
			}
			logs, err := drv.TailLogs(100)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), logs)
			return nil
		},
	}
}
