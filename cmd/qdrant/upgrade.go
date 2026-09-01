package qdrant

import (
	"OpsVault/cmd/common"
	"OpsVault/internal/driver"

	"github.com/spf13/cobra"
)

func (c *commandSet) newUpgradeCommand() *cobra.Command {
	var tag string
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade Qdrant service image",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver("")
			if err != nil {
				return err
			}
			return drv.Upgrade(tag)
		},
	}
	cmd.Flags().StringVar(&tag, "tag", "", "Target image tag (e.g. v1.13.0)")
	_ = cmd.MarkFlagRequired("tag")
	return cmd
}

func (c *commandSet) newLogCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "log",
		Short: "Show Qdrant service logs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			drv, err := c.driver("")
			if err != nil {
				return err
			}
			out, err := drv.TailLogs(100)
			cmd.Print(out)
			return err
		},
	}
}
