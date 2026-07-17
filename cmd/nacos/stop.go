package nacos

import (
	"fmt"

	"OpsVault/cmd/common"
	"OpsVault/internal/driver"
	"OpsVault/pkg/logger"

	"github.com/spf13/cobra"
)

func (c *commandSet) newStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop Nacos",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver("")
			if err != nil {
				return err
			}
			if err := drv.Stop(); err != nil {
				logger.AuditLog("nacos", "stop", "", false)
				return err
			}
			logger.AuditLog("nacos", "stop", "", true)
			fmt.Println("Nacos stopped successfully.")
			return nil
		},
	}
}
