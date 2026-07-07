package redis

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
			if err := drv.Install(); err != nil {
				return err
			}
			port := c.config.GetString("redis.port")
			if port == "" {
				port = "6379"
			}
			creds := []credutil.Credential{
				{Label: "主机", Value: fmt.Sprintf("localhost:%s", port)},
			}
			if password != "" {
				creds = append(creds, credutil.Credential{Label: "密  码", Value: password})
			} else {
				creds = append(creds, credutil.Credential{Label: "密  码", Value: "(无认证)"})
			}
			credutil.PrintCredentials("Redis", creds)
			return nil
		},
	}
	cmd.Flags().StringVar(&password, "pwd", "", "Redis password (leave empty for no auth)")
	cmd.Flags().BoolVar(&randomPwd, "random-pwd", false, "Generate a secure random password")
	return cmd
}

