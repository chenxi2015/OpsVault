package elk

import (
	"OpsVault/cmd/common"
	"OpsVault/internal/driver"

	"github.com/spf13/cobra"
)

func (c *commandSet) newUpgradeCommand() *cobra.Command {
	var version string
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade ELK Stack version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver()
			if err != nil {
				return err
			}
			return drv.Upgrade(version)
		},
	}
	cmd.Flags().StringVar(&version, "tag", "8.12.0", "Target Docker image tag version (e.g. 8.12.0)")
	_ = cmd.MarkFlagRequired("tag")
	return cmd
}
