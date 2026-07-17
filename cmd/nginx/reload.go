package nginx

import "github.com/spf13/cobra"

// newReloadCommand creates a subcommand to reload Nginx configuration without restarting the service.
func (c *commandSet) newReloadCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reload",
		Short: "Reload Nginx configuration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.driver().Reload()
		},
	}
}
