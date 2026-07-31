package redis

import (
	"fmt"

	"OpsVault/cmd/common"
	"OpsVault/internal/driver"
	"OpsVault/pkg/credutil"
	"OpsVault/pkg/logger"

	"github.com/spf13/cobra"
)

func (c *commandSet) newInstallCommand() *cobra.Command {
	var (
		password  string
		randomPwd bool
	)
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Redis",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			if randomPwd {
				password = credutil.GenPassword(20)
			} else if password == "" {
				password = c.config.GetString("redis.password")
			}
			drv, err := c.driver(password)
			if err != nil {
				return err
			}
			logger.Infof("Installing Redis service...")
			if err := drv.Install(); err != nil {
				logger.AuditLog("redis", "install", fmt.Sprintf("image=%s", c.config.GetString("redis.image")), false)
				return err
			}
			logger.AuditLog("redis", "install", fmt.Sprintf("image=%s", c.config.GetString("redis.image")), true)
			credutil.PrintCredentials("Redis", drv.GetCredentials())
			return nil
		},
	}
	cmd.Flags().StringVar(&password, "pwd", "", "Redis password (leave empty for no auth)")
	cmd.Flags().BoolVar(&randomPwd, "random-pwd", false, "Generate a secure random password")
	return cmd
}

