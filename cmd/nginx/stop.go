package nginx

import "github.com/spf13/cobra"

func (c *commandSet) newStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop Nginx",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.driver().Stop()
		},
	}
}
