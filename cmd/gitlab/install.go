package gitlab

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
		Short: "Install GitLab in Docker mode",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver()
			if err != nil {
				return err
			}
			if err := drv.Install(); err != nil {
				logger.AuditLog("gitlab", "install", fmt.Sprintf("image=%s", c.config.GetString("gitlab.image")), false)
				return err
			}
			logger.AuditLog("gitlab", "install", fmt.Sprintf("image=%s", c.config.GetString("gitlab.image")), true)
			credutil.PrintCredentials("GitLab", drv.GetCredentials())
			return nil
		},
	}
	return cmd
}
