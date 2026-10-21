package nginx

import (
	"OpsVault/internal/driver/binary"

	"github.com/spf13/cobra"
)

func (c *commandSet) newLogCommand() *cobra.Command {
	var (
		logType string
		domain  string
		lines   int
		follow  bool
	)

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Tail Nginx access, error, or systemd logs",
		Long: `Tail Nginx logs. By default, displays the access log (access_nginx.log or domain_access.log).
Use -t/--type to switch between access, error, and systemd logs.
Use -d/--domain to specify a virtual host log.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			drv := c.driver()
			opts := binary.NginxLogOptions{
				Type:   logType,
				Domain: domain,
				Lines:  lines,
				Follow: follow,
			}
			return drv.TailServiceLogs(opts, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVarP(&logType, "type", "t", "access", "Log type to view (access, error, systemd)")
	cmd.Flags().StringVarP(&domain, "domain", "d", "", "Virtual host domain name (e.g. example.com)")
	cmd.Flags().IntVarP(&lines, "lines", "n", 100, "Number of lines to show")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output in real time")

	rotateCmd := &cobra.Command{
		Use:   "rotate",
		Short: "Manually trigger Nginx log rotation now",
		RunE: func(cmd *cobra.Command, _ []string) error {
			drv := c.driver()
			if err := drv.RotateLogs(); err != nil {
				return err
			}
			cmd.Println("Nginx log rotation triggered successfully.")
			return nil
		},
	}
	cmd.AddCommand(rotateCmd)

	return cmd
}
