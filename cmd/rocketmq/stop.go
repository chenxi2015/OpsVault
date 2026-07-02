package rocketmq

import (
	"OpsVault/cmd/common"
	"OpsVault/internal/driver"

	"github.com/spf13/cobra"
)

func (c *commandSet) newStopCommand() *cobra.Command {
	return &cobra.Command{Use: "stop", Short: "Stop RocketMQ", RunE: func(cmd *cobra.Command, _ []string) error {
		if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
			return err
		}
		drv, err := c.driver()
		if err != nil {
			return err
		}
		return drv.Stop()
	}}
}
