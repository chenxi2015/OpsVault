package nginx

import (
	"os/exec"

	"github.com/spf13/cobra"
)

func (c *commandSet) newLogCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "log",
		Short: "Tail Nginx systemd logs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			journal := exec.Command("journalctl", "-u", "nginx", "-n", "100", "--no-pager")
			out, err := journal.CombinedOutput()
			cmd.Print(string(out))
			return err
		},
	}
}
