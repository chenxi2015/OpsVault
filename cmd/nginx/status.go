package nginx

import (
	"OpsVault/cmd/common"

	"github.com/spf13/cobra"
)

func (c *commandSet) newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show Nginx status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			status, err := c.driver().Status()
			if err != nil {
				return err
			}
			common.PrintStatus(cmd, status)
			return nil
		},
	}
}
