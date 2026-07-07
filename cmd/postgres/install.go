package postgres

import (
	"fmt"

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
			if password == "" {
				return fmt.Errorf("PostgreSQL password is required: use --pwd <pwd> or --random-pwd")
			}
			drv, err := c.driver(password)
			if err != nil {
				return err
			}
			if err := drv.Install(); err != nil {
				return err
			}
			port := c.config.GetString("postgres.port")
			if port == "" {
				port = "5432"
			}
			credutil.PrintCredentials("PostgreSQL", []credutil.Credential{
				{Label: "主机", Value: fmt.Sprintf("localhost:%s", port)},
				{Label: "用户名", Value: "postgres"},
				{Label: "密  码", Value: password},
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&password, "pwd", "", "PostgreSQL password")
	cmd.Flags().BoolVar(&randomPwd, "random-pwd", false, "Generate a secure random password")
	return cmd
}

