package minio

import (
	"OpsVault/cmd/common"
	"OpsVault/internal/driver"
	"OpsVault/pkg/credutil"

	"github.com/spf13/cobra"
)

func (c *commandSet) newInstallCommand() *cobra.Command {
	var (
		password  string
		randomPwd bool
	)
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install MinIO",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			if randomPwd {
				password = credutil.GenPassword(20)
			} else if password == "" {
				password = c.config.GetString("minio.root_password")
			}
			drv, err := c.driver(password)
			if err != nil {
				return err
			}
			if err := drv.Install(); err != nil {
				return err
			}
			credutil.PrintCredentials("MinIO", drv.GetCredentials())
			return nil
		},
	}
	cmd.Flags().StringVar(&password, "pwd", "", "MinIO root password")
	cmd.Flags().BoolVar(&randomPwd, "random-pwd", false, "Generate a secure random root password")
	return cmd
}
