package jenkins

import (
	"fmt"

	"OpsVault/cmd/common"
	"OpsVault/internal/driver"
	"OpsVault/pkg/logger"

	"github.com/spf13/cobra"
)

func (c *commandSet) newUninstallCommand() *cobra.Command {
	var purge bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Jenkins",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver()
			if err != nil {
				return err
			}
			if err := drv.Uninstall(purge); err != nil {
				logger.AuditLog("jenkins", "uninstall", fmt.Sprintf("purge=%v", purge), false)
				return err
			}
			logger.AuditLog("jenkins", "uninstall", fmt.Sprintf("purge=%v", purge), true)
			return nil
		},
	}
	cmd.Flags().BoolVar(&purge, "purge", false, "delete data directory")
	return cmd
}
