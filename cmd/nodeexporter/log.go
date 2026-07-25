package nodeexporter

import (
	"OpsVault/cmd/common"
	"OpsVault/internal/driver"

	"github.com/spf13/cobra"
)

func (c *commandSet) newLogCommand() *cobra.Command {
	var lines int
	cmd := &cobra.Command{
		Use:   "log",
		Short: "View logs of Node Exporter container",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver()
			if err != nil {
				return err
			}
			out, err := drv.TailLogs(lines)
			if err != nil {
				return err
			}
			cmd.Println(out)
			return nil
		},
	}
	cmd.Flags().IntVarP(&lines, "lines", "n", 100, "Number of log lines to tail")
	return cmd
}
