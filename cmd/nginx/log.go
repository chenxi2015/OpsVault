package nginx

import (
	"github.com/spf13/cobra"
)

func (c *commandSet) newLogCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "log",
		Short: "Tail Nginx systemd logs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := c.driver().TailLogs(100)
			cmd.Print(out)
			return err
		},
	}
}

