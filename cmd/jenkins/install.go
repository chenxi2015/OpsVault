package jenkins

import (
	"fmt"

	"OpsVault/cmd/common"
	"OpsVault/internal/driver"
	"OpsVault/pkg/credutil"
	"OpsVault/pkg/logger"

	"github.com/spf13/cobra"
)

func (c *commandSet) newInstallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Jenkins in Docker mode",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver()
			if err != nil {
				return err
			}
			if err := drv.Install(); err != nil {
				logger.AuditLog("jenkins", "install", fmt.Sprintf("image=%s", c.config.GetString("jenkins.image")), false)
				return err
			}
			logger.AuditLog("jenkins", "install", fmt.Sprintf("image=%s", c.config.GetString("jenkins.image")), true)
			credutil.PrintCredentials("Jenkins", drv.GetCredentials())
			return nil
		},
	}
	return cmd
}
