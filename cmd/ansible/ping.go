package ansiblecmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func (c *commandSet) newPingCommand() *cobra.Command {
	var group string
	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Ping target hosts to check connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			exec, cleanup, err := c.getExecutor()
			if err != nil {
				return err
			}
			defer cleanup()

			fmt.Printf("Pinging hosts in group: %s...\n", group)
			err = exec.RunAnsible(cmd.Context(), group, "ping", "", os.Stdout, os.Stderr)
			if err != nil {
				return fmt.Errorf("ping command failed: %w", err)
			}
			fmt.Println("Ping completed successfully.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&group, "group", "g", "all", "target host group to ping")
	return cmd
}
