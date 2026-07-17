package ansiblecmd

import (
	"bytes"
	"fmt"

	"OpsVault/internal/driver/ansible"

	"github.com/charmbracelet/lipgloss"
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

			var stdoutBuf bytes.Buffer
			var stderrBuf bytes.Buffer

			_ = exec.RunAnsible(cmd.Context(), group, "ping", "", &stdoutBuf, &stderrBuf)

			results := ansible.ParsePingOutput(stdoutBuf.String())
			if len(results) == 0 {
				fmt.Println("No ping results returned from remote hosts.")
				if stderrBuf.Len() > 0 {
					fmt.Printf("Error logs:\n%s\n", stderrBuf.String())
				}
				return fmt.Errorf("ping command failed")
			}

			// Render Table Headers
			ipHeader := headerStyle.Width(20).Render("HOST IP")
			statusHeader := headerStyle.Width(15).Render("STATUS")
			messageHeader := headerStyle.Width(60).Render("MESSAGE")

			cmd.Println("\n" + lipgloss.JoinHorizontal(lipgloss.Top, ipHeader, statusHeader, messageHeader))

			hasFailed := false
			for _, r := range results {
				ipRow := rowStyle.Width(20).Render(r.IP)

				var statusVal string
				switch r.Status {
				case "SUCCESS", "CHANGED":
					statusVal = successText.Render(r.Status)
				case "FAILED":
					statusVal = failText.Render(r.Status)
					hasFailed = true
				default:
					statusVal = failText.Render(r.Status)
					hasFailed = true
				}
				statusRow := rowStyle.Width(15).Render(statusVal)
				messageRow := rowStyle.Width(60).Render(r.Message)

				cmd.Println(lipgloss.JoinHorizontal(lipgloss.Top, ipRow, statusRow, messageRow))
			}
			cmd.Println()

			if hasFailed {
				return fmt.Errorf("some hosts failed to ping")
			}
			cmd.Println("Ping completed successfully.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&group, "group", "g", "all", "target host group to ping")
	return cmd
}

