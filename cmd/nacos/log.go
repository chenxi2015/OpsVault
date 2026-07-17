package nacos

import (
	"github.com/spf13/cobra"
)

func (c *commandSet) newLogCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "log",
		Short: "Show Nacos logs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			drv, err := c.driver("")
			if err != nil {
				return err
			}
			out, err := drv.TailLogs(100)
			cmd.Print(out)
			return err
		},
	}
}
