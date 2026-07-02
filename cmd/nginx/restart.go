package nginx

import "github.com/spf13/cobra"

func (c *commandSet) newRestartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart Nginx",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.driver().Restart()
		},
	}
}
