package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"OpsVault/pkg/logger"
	"OpsVault/pkg/selfupdate"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the current OpsVault version and platform info",
		Run: func(cmd *cobra.Command, _ []string) {
			card := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("14")).
				Padding(0, 2).
				Render(fmt.Sprintf(
					"%s  %s\n%s  %s/%s\n%s  %s",
					infoStyle.Render("Version:   "),
					successStyle.Render(Version),
					infoStyle.Render("Platform:  "),
					runtime.GOOS, runtime.GOARCH,
					infoStyle.Render("Go Runtime:"),
					runtime.Version(),
				))
			cmd.Println(card)
		},
	}
}

func newUpgradeCommand(cfg *viper.Viper) *cobra.Command {
	var (
		tag       string
		checkOnly bool
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Self-upgrade OpsVault to the latest or a specific release version",
		Example: `  # Upgrade to the latest stable version
  opsvault upgrade

  # Upgrade or rollback to a specific version
  opsvault upgrade --tag v1.0.0

  # Only check if a newer version is available
  opsvault upgrade --check`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var proxies []string
			if cfg != nil {
				proxies = cfg.GetStringSlice("system.github_proxies")
			}

			updater := selfupdate.New(selfupdate.Options{
				CurrentVersion: Version,
				TargetTag:      tag,
				CheckOnly:      checkOnly,
				Force:          force,
				Proxies:        proxies,
			})

			cmd.Printf("%s Current version: %s (%s/%s)\n",
				infoStyle.Render("->"),
				headerStyle.Render(Version),
				runtime.GOOS, runtime.GOARCH,
			)

			res, err := updater.Run()
			if err != nil {
				logger.AuditLog("selfupdate", "upgrade", err.Error(), false)
				return fmt.Errorf("opsvault self-upgrade failed: %w", err)
			}

			if res.AlreadyLatest {
				cmd.Println(successStyle.Render(fmt.Sprintf("✓ OpsVault is already at the target version (%s).", res.TargetVersion)))
				return nil
			}

			if checkOnly {
				cmd.Println(warnStyle.Render(fmt.Sprintf("★ New version available: %s -> %s", res.CurrentVersion, res.TargetVersion)))
				if notes := strings.TrimSpace(res.ReleaseNotes); notes != "" {
					cmd.Printf("\n%s\n%s\n", titleStyle.Render("Release Notes:"), notes)
				}
				cmd.Printf("\nRun %s to upgrade now.\n", successStyle.Render("opsvault upgrade"))
				return nil
			}

			logger.AuditLog("selfupdate", "upgrade", fmt.Sprintf("%s -> %s (%s)", res.CurrentVersion, res.TargetVersion, res.ExecutablePath), true)

			summary := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("10")).
				Padding(0, 2).
				Render(fmt.Sprintf(
					"%s\n%s %s -> %s\n%s %s",
					successStyle.Render("✓ OpsVault upgraded successfully!"),
					infoStyle.Render("Version:"),
					res.CurrentVersion,
					successStyle.Render(res.TargetVersion),
					infoStyle.Render("Binary: "),
					res.ExecutablePath,
				))
			cmd.Println("\n" + summary)
			return nil
		},
	}

	cmd.Flags().StringVar(&tag, "tag", "", "Target version tag to install (e.g. v1.0.0, defaults to latest)")
	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates without downloading or replacing binary")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force reinstall even if already on the target version")

	return cmd
}

