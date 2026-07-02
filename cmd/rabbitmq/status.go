package rabbitmq

import (
	"OpsVault/cmd/common"

	"github.com/spf13/cobra"
)

func (c *commandSet) newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show RabbitMQ status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			drv, err := c.driver("", "")
			if err != nil {
				return err
			}
			status, err := drv.Status()
			if err != nil {
				return err
			}
			common.PrintStatus(cmd, status)
			return nil
		},
	}
}
