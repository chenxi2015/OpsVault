package nacos

import (
	"fmt"

	"OpsVault/cmd/common"
	"OpsVault/internal/driver"
	"OpsVault/pkg/logger"

	"github.com/spf13/cobra"
)

func (c *commandSet) newRestartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart Nacos",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver("")
			if err != nil {
				return err
			}
			if err := drv.Restart(); err != nil {
				logger.AuditLog("nacos", "restart", "", false)
				return err
			}
			logger.AuditLog("nacos", "restart", "", true)
			fmt.Println("Nacos restarted successfully.")
			return nil
		},
	}
}
