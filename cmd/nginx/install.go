package nginx

import "github.com/spf13/cobra"

func (c *commandSet) newInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install Nginx from source",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.driver().Install()
		},
	}
}
