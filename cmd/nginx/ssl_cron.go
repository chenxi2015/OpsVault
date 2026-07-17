package nginx

import (
	"fmt"
	"os"
	"path/filepath"

	"OpsVault/pkg/logger"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	sslCronFile    = "/etc/cron.d/opsvault-ssl-renew"
	sslCronComment = "# Managed by OpsVault — auto SSL certificate renewal"
)

func newSSLCronCommand(cfg *viper.Viper, opsvaultBin string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cron",
		Short: "Manage automatic SSL certificate renewal cron job",
	}
	cmd.AddCommand(
		newSSLCronEnableCommand(cfg, opsvaultBin),
		newSSLCronDisableCommand(),
		newSSLCronStatusCommand(),
	)
	return cmd
}

func newSSLCronEnableCommand(cfg *viper.Viper, opsvaultBin string) *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Register a monthly cron job to auto-renew all SSL certificates",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opsvaultBin == "" {
				bin, err := os.Executable()
				if err != nil {
					return fmt.Errorf("cannot detect opsvault binary path: %w", err)
				}
				opsvaultBin = bin
			}
			logDir := cfg.GetString("log.storage_path")
			if logDir == "" {
				logDir = "/data/opsvault/logs"
			}
			logFile := filepath.Join(logDir, "ssl-renew.log")

			content := fmt.Sprintf("%s\n0 3 1 * * root %s nginx ssl renew >> %s 2>&1\n",
				sslCronComment,
				opsvaultBin,
				logFile,
			)
			if err := os.WriteFile(sslCronFile, []byte(content), 0o644); err != nil {
				return fmt.Errorf("write cron file %s: %w", sslCronFile, err)
			}
			logger.AuditLog("nginx", "ssl-cron-enable", "cron=monthly", true)
			cmd.Printf("✓ SSL auto-renewal cron registered: %s\n", sslCronFile)
			cmd.Println("  Schedule: every 1st day of month at 03:00")
			cmd.Printf("  Log: %s\n", logFile)
			return nil
		},
	}
}

func newSSLCronDisableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Remove the SSL auto-renewal cron job",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := os.Remove(sslCronFile); err != nil {
				if os.IsNotExist(err) {
					cmd.Println("SSL auto-renewal cron is not enabled.")
					return nil
				}
				return fmt.Errorf("remove cron file: %w", err)
			}
			logger.AuditLog("nginx", "ssl-cron-disable", "", true)
			cmd.Printf("✓ SSL auto-renewal cron removed: %s\n", sslCronFile)
			return nil
		},
	}
}

func newSSLCronStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current SSL auto-renewal cron status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			data, err := os.ReadFile(sslCronFile)
			if err != nil {
				if os.IsNotExist(err) {
					cmd.Println("SSL auto-renewal cron: DISABLED")
					return nil
				}
				return fmt.Errorf("read cron file: %w", err)
			}
			cmd.Printf("SSL auto-renewal cron: ENABLED (%s)\n\n", sslCronFile)
			cmd.Printf("--- cron content ---\n%s\n", string(data))
			return nil
		},
	}
}
