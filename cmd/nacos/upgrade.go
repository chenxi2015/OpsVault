package nacos

import (
	"fmt"

	"OpsVault/cmd/common"
	"OpsVault/internal/driver"
	"OpsVault/pkg/logger"

	"github.com/spf13/cobra"
)

func (c *commandSet) newUpgradeCommand() *cobra.Command {
	var tag string
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade Nacos docker image tag",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver("")
			if err != nil {
				return err
			}
			if err := drv.Upgrade(tag); err != nil {
				logger.AuditLog("nacos", "upgrade", "tag="+tag, false)
				return err
			}
			logger.AuditLog("nacos", "upgrade", "tag="+tag, true)
			fmt.Printf("Nacos upgraded to version %s successfully.\n", tag)
			return nil
		},
	}
	cmd.Flags().StringVar(&tag, "tag", "v2.3.2", "target docker image tag")
	_ = cmd.MarkFlagRequired("tag")
	return cmd
}
