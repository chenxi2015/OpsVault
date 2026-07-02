package nginx

import "github.com/spf13/cobra"

func (c *commandSet) newUpgradeCommand() *cobra.Command {
	var target string
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade Nginx",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.driver().Upgrade(target)
		},
	}
	cmd.Flags().StringVar(&target, "tag", "", "target version")
	return cmd
}
