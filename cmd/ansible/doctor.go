package ansiblecmd

import (
	"bytes"
	"fmt"
	"strings"

	"OpsVault/internal/driver/ansible"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("39")).
			PaddingRight(2)

	rowStyle = lipgloss.NewStyle().
			PaddingRight(2)

	successText = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green
	warnText    = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // Yellow
	failText    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // Red
)

func (c *commandSet) newDoctorCommand() *cobra.Command {
	var group string
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Batch inspect remote host status and resource usages",
		RunE: func(cmd *cobra.Command, args []string) error {
			exec, cleanup, err := c.getExecutor()
			if err != nil {
				return err
			}
			defer cleanup()

			fmt.Printf("Inspecting remote hosts in group: %s...\n", group)

			// Combine multiple inspection commands to parse
			inspectionCmd := `echo "===UPTIME==="; uptime; echo "===FREE==="; free -m; echo "===DF==="; df -h /; echo "===SERVICES==="; systemctl is-active docker || echo "inactive"; systemctl is-active nginx || echo "inactive"`

			var stdoutBuf bytes.Buffer
			var stderrBuf bytes.Buffer

			// Run ansible ad-hoc command silently to gather information
			_ = exec.RunAnsible(cmd.Context(), group, "shell", inspectionCmd, &stdoutBuf, &stderrBuf)

			results := ansible.ParseDoctorOutput(stdoutBuf.String())
			if len(results) == 0 {
				fmt.Println("No inspection results returned from remote hosts.")
				if stderrBuf.Len() > 0 {
					fmt.Printf("Error logs:\n%s\n", stderrBuf.String())
				}
				return nil
			}

			// Render Table Headers
			ipHeader := headerStyle.Width(18).Render("HOST IP")
			statusHeader := headerStyle.Width(12).Render("STATUS")
			uptimeHeader := headerStyle.Width(15).Render("UPTIME")
			memHeader := headerStyle.Width(20).Render("MEMORY (USED/TOTAL)")
			diskHeader := headerStyle.Width(15).Render("DISK (USED/TOTAL)")
			dockerHeader := headerStyle.Width(10).Render("DOCKER")
			nginxHeader := headerStyle.Width(10).Render("NGINX")

			cmd.Println("\n" + lipgloss.JoinHorizontal(lipgloss.Top, ipHeader, statusHeader, uptimeHeader, memHeader, diskHeader, dockerHeader, nginxHeader))

			// Render Rows
			for _, r := range results {
				ipRow := rowStyle.Width(18).Render(r.IP)

				var statusVal string
				switch r.Status {
				case "SUCCESS", "CHANGED":
					statusVal = successText.Render(r.Status)
				case "FAILED":
					statusVal = failText.Render(r.Status)
				default:
					statusVal = failText.Render(r.Status)
				}
				statusRow := rowStyle.Width(12).Render(statusVal)

				if r.Status == "UNREACHABLE" || r.Status == "FAILED" {
					emptyRow := rowStyle.Render(failText.Render("N/A (Connection Failed)"))
					cmd.Println(lipgloss.JoinHorizontal(lipgloss.Top, ipRow, statusRow, emptyRow))
					continue
				}

				uptimeRow := rowStyle.Width(15).Render(r.Uptime)

				memStr := fmt.Sprintf("%s / %s", r.MemUsed, r.MemTotal)
				if r.MemUsed == "" || r.MemTotal == "" {
					memStr = "N/A"
				}
				memRow := rowStyle.Width(20).Render(memStr)

				diskStr := fmt.Sprintf("%s (%s)", r.DiskSize, r.DiskUsePct)
				if r.DiskSize == "" {
					diskStr = "N/A"
				}
				diskRow := rowStyle.Width(15).Render(diskStr)

				var dockerVal string
				if strings.Contains(r.DockerState, "active") {
					dockerVal = successText.Render("active")
				} else {
					dockerVal = warnText.Render("inactive")
				}
				dockerRow := rowStyle.Width(10).Render(dockerVal)

				var nginxVal string
				if strings.Contains(r.NginxState, "active") {
					nginxVal = successText.Render("active")
				} else {
					nginxVal = warnText.Render("inactive")
				}
				nginxRow := rowStyle.Width(10).Render(nginxVal)

				cmd.Println(lipgloss.JoinHorizontal(lipgloss.Top, ipRow, statusRow, uptimeRow, memRow, diskRow, dockerRow, nginxRow))
			}
			cmd.Println()

			return nil
		},
	}
	cmd.Flags().StringVarP(&group, "group", "g", "all", "target host group to inspect")
	return cmd
}
