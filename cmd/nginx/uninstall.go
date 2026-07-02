package nginx

import "github.com/spf13/cobra"

func (c *commandSet) newUninstallCommand() *cobra.Command {
	var purge bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Nginx",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.driver().Uninstall(purge)
		},
	}
	cmd.Flags().BoolVar(&purge, "purge", false, "delete website and ssl data")
	return cmd
}
