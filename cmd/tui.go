package cmd

import (
	"OpsVault/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func newTUICommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the OpsVault terminal UI",
		RunE: func(cmd *cobra.Command, _ []string) error {
			provider := tui.NewRuntimeStatusProvider(AppConfig(), DockerClient)
			program := tea.NewProgram(tui.NewRootModel(provider), tea.WithAltScreen())
			_, err := program.Run()
			return err
		},
	}
}
