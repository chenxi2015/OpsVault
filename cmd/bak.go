package cmd

import (
	"fmt"
	"strings"

	"OpsVault/internal/backup"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	bakHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("39")).
			PaddingRight(2)

	bakRowStyle = lipgloss.NewStyle().
			PaddingRight(2)

	bakSuccessCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("10")).
			Padding(1, 3).
			MarginTop(1).
			MarginBottom(1)
)

func newBakCommand(cfg *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bak",
		Short: "Manage configuration backups",
		Long:  "Backup and restore configuration files for Nginx, MySQL, Redis, RocketMQ, RabbitMQ, PostgreSQL, Nacos and global default.yaml",
	}

	cmd.AddCommand(
		newBakCreateCommand(cfg),
		newBakListCommand(cfg),
		newBakRestoreCommand(cfg),
		newBakDeleteCommand(cfg),
	)

	return cmd
}

func newBakCreateCommand(cfg *viper.Viper) *cobra.Command {
	var name string
	var desc string

	cmd := &cobra.Command{
		Use:   "create [service]",
		Short: "Create a configuration backup",
		Long:  "Create a backup of service configuration files. You can specify a single service (nginx, mysql, redis, nacos, etc.) or omit/use 'all' to back up all configurations.",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := backup.NewBackupManager(cfg)

			var services []string
			if len(args) > 0 {
				services = args
			} else {
				services = []string{"all"}
			}

			meta, err := manager.CreateBackup(services, name, desc)
			if err != nil {
				return err
			}

			// Display beautiful success message
			sizeStr := formatBytes(meta.SizeBytes)
			servicesStr := strings.Join(meta.Services, ", ")

			labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
			valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
			servicesValueStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))

			cardContent := fmt.Sprintf(
				"%s\n\n%s  %s\n%s  %s\n%s  %s\n%s  %s\n%s  %s",
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("✓ Backup Created Successfully"),
				labelStyle.Render("Name:       "), valueStyle.Render(meta.Name),
				labelStyle.Render("Services:   "), servicesValueStyle.Render(servicesStr),
				labelStyle.Render("Directory:  "), valueStyle.Render(manager.GetBackupDir()),
				labelStyle.Render("Size:       "), valueStyle.Render(sizeStr),
				labelStyle.Render("Time:       "), valueStyle.Render(meta.Timestamp.Format("2006-01-02 15:04:05")),
			)
			if meta.Description != "" {
				cardContent += fmt.Sprintf("\n%s  %s", labelStyle.Render("Desc:       "), valueStyle.Render(meta.Description))
			}

			cmd.Println(bakSuccessCard.Render(cardContent))
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Custom backup name (defaults to auto-generated timestamp)")
	cmd.Flags().StringVarP(&desc, "desc", "d", "", "Backup description")

	return cmd
}

func newBakListCommand(cfg *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configuration backups",
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager := backup.NewBackupManager(cfg)
			backups, err := manager.ListBackups()
			if err != nil {
				return err
			}

			if len(backups) == 0 {
				cmd.Println("No backups found in storage path:", manager.GetBackupDir())
				return nil
			}

			// Render Table Headers
			nameHeader := bakHeaderStyle.Width(25).Render("NAME")
			sizeHeader := bakHeaderStyle.Width(10).Render("SIZE")
			servicesHeader := bakHeaderStyle.Width(20).Render("SERVICES")
			timeHeader := bakHeaderStyle.Width(22).Render("CREATED TIME")
			descHeader := bakHeaderStyle.Width(30).Render("DESCRIPTION")

			cmd.Println(lipgloss.JoinHorizontal(lipgloss.Top, nameHeader, sizeHeader, servicesHeader, timeHeader, descHeader))

			// Render Rows
			for _, b := range backups {
				nameRow := bakRowStyle.Width(25).Render(b.Name)
				sizeRow := bakRowStyle.Width(10).Render(formatBytes(b.SizeBytes))
				servicesRow := bakRowStyle.Width(20).Render(strings.Join(b.Services, ", "))
				timeRow := bakRowStyle.Width(22).Render(b.Timestamp.Format("2006-01-02 15:04:05"))
				descRow := bakRowStyle.Width(30).Render(b.Description)

				cmd.Println(lipgloss.JoinHorizontal(lipgloss.Top, nameRow, sizeRow, servicesRow, timeRow, descRow))
			}

			return nil
		},
	}
}

func newBakRestoreCommand(cfg *viper.Viper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "restore <backup_name> [service]",
		Short: "Restore configuration from a backup",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			var service string
			if len(args) > 1 {
				service = args[1]
			}

			if !force {
				var targetMsg string
				if service != "" {
					targetMsg = fmt.Sprintf("service %s", service)
				} else {
					targetMsg = "all services in the backup"
				}

				cmd.Printf("Warning: This will overwrite current configurations for %s. Continue? [y/N]: ", targetMsg)
				var response string
				_, _ = fmt.Scanln(&response)
				response = strings.ToLower(strings.TrimSpace(response))
				if response != "y" && response != "yes" {
					cmd.Println("Restoration cancelled.")
					return nil
				}
			}

			manager := backup.NewBackupManager(cfg)
			if err := manager.RestoreBackup(name, service); err != nil {
				return err
			}

			cmd.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("\n✓ Configurations restored successfully!"))
			cmd.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("Note: Please restart modified services to apply new configurations."))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force restoration without interactive confirmation")

	return cmd
}

func newBakDeleteCommand(cfg *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <backup_name>",
		Short: "Delete a configuration backup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			manager := backup.NewBackupManager(cfg)
			if err := manager.DeleteBackup(name); err != nil {
				return err
			}
			cmd.Printf("Backup %s successfully deleted.\n", name)
			return nil
		},
	}
}

func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(bytes)/1024.0)
	} else {
		return fmt.Sprintf("%.2f MB", float64(bytes)/(1024.0*1024.0))
	}
}
