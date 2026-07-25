package nodeexporter

import (
	"OpsVault/cmd/common"
	"OpsVault/internal/driver"

	"github.com/spf13/cobra"
)

func (c *commandSet) newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Display status of Node Exporter service",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver()
			if err != nil {
				return err
			}
			st, err := drv.Status()
			if err != nil {
				return err
			}
			common.PrintStatus(cmd, st)
			return nil
		},
	}
}
