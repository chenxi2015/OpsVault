package postgres

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
		Short: "Install PostgreSQL",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			if randomPwd {
				password = credutil.GenPassword(20)
			} else if password == "" {
				password = c.config.GetString("postgres.password")
			}
			drv, err := c.driver(password)
			if err != nil {
				return err
			}
			logger.Infof("Installing PostgreSQL service...")
			if err := drv.Install(); err != nil {
				logger.AuditLog("postgres", "install", fmt.Sprintf("image=%s", c.config.GetString("postgres.image")), false)
				return err
			}
			logger.AuditLog("postgres", "install", fmt.Sprintf("image=%s", c.config.GetString("postgres.image")), true)
			credutil.PrintCredentials("PostgreSQL", drv.GetCredentials())
			return nil
		},
	}
	cmd.Flags().StringVar(&password, "pwd", "", "PostgreSQL password")
	cmd.Flags().BoolVar(&randomPwd, "random-pwd", false, "Generate a secure random password")
	return cmd
}

