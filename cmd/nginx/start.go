package nginx

import "github.com/spf13/cobra"

func (c *commandSet) newStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start Nginx",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.driver().Start()
		},
	}
}
