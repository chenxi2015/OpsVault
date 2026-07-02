package mysql

import "github.com/spf13/cobra"

func (c *commandSet) newRestartCommand() *cobra.Command {
	return lifecycleCommand(c, "restart", "Restart MySQL", func(d dockerDriverShim) error { return d.Restart() })
}
