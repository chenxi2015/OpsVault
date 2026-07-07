package nginx

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func (c *commandSet) newInstallCommand() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Nginx from source",
		RunE: func(cmd *cobra.Command, _ []string) error {
			drv := c.driver()
			if !force {
				if exists, reason := drv.CheckExisting(); exists {
					cmd.Printf("Warning: %s.\n", reason)
					cmd.Printf("Are you sure you want to reinstall/overwrite Nginx? [y/N]: ")
					var response string
					_, _ = fmt.Scanln(&response)
					response = strings.ToLower(strings.TrimSpace(response))
					if response != "y" && response != "yes" {
						cmd.Println("Installation cancelled.")
						return nil
					}
				}
			}
			return drv.Install()
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force installation and overwrite existing Nginx")
	return cmd
}
