package mysql

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
		rootPwd   string
		randomPwd bool
	)
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install MySQL in Docker mode",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			if randomPwd {
				rootPwd = credutil.GenPassword(20)
			} else if rootPwd == "" {
				rootPwd = c.config.GetString("mysql.root_password")
			}
			if rootPwd == "" {
				return fmt.Errorf("MySQL root password is required: use --root-pwd <pwd> or --random-pwd")
			}
			drv, err := c.driver(rootPwd)
			if err != nil {
				return err
			}
			if err := drv.Install(); err != nil {
				logger.AuditLog("mysql", "install", fmt.Sprintf("image=%s", c.config.GetString("mysql.image")), false)
				return err
			}
			logger.AuditLog("mysql", "install", fmt.Sprintf("image=%s", c.config.GetString("mysql.image")), true)
			port := c.config.GetString("mysql.port")
			if port == "" {
				port = "3306"
			}
			credutil.PrintCredentials("MySQL", []credutil.Credential{
				{Label: "主机", Value: fmt.Sprintf("localhost:%s", port)},
				{Label: "用户名", Value: "root"},
				{Label: "密  码", Value: rootPwd},
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&rootPwd, "root-pwd", "", "MySQL root password")
	cmd.Flags().BoolVar(&randomPwd, "random-pwd", false, "Generate a secure random password")
	return cmd
}

