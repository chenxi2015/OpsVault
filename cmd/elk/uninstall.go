package elk

import (
	"OpsVault/cmd/common"
	"OpsVault/internal/driver"

	"github.com/spf13/cobra"
)

func (c *commandSet) newUninstallCommand() *cobra.Command {
	var purge bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall ELK Stack",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver()
			if err != nil {
				return err
			}
			return drv.Uninstall(purge)
		},
	}
	cmd.Flags().BoolVar(&purge, "purge", false, "Purge configuration and data directories")
	return cmd
}
