package mysql

import "github.com/spf13/cobra"

func (c *commandSet) newStopCommand() *cobra.Command {
	return lifecycleCommand(c, "stop", "Stop MySQL", func(d dockerDriverShim) error { return d.Stop() })
}
