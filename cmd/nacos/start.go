package nacos

import (
	"fmt"

	"OpsVault/cmd/common"
	"OpsVault/internal/driver"
	"OpsVault/pkg/logger"

	"github.com/spf13/cobra"
)

func (c *commandSet) newStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start Nacos",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver("")
			if err != nil {
				return err
			}
			if err := drv.Start(); err != nil {
				logger.AuditLog("nacos", "start", "", false)
				return err
			}
			logger.AuditLog("nacos", "start", "", true)
			fmt.Println("Nacos started successfully.")
			return nil
		},
	}
}
