package cmd

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"OpsVault/internal/tui"
	"OpsVault/pkg/logger"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func newTUICommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the OpsVault terminal UI",
		RunE: func(cmd *cobra.Command, _ []string) error {
			provider := tui.NewRuntimeStatusProvider(AppConfig(), DockerClient)
			model := tui.NewRootModel(provider)
			program := tea.NewProgram(&model, tea.WithAltScreen())

			// Determine log directory and file
			logDir := AppConfig().GetString("log.storage_path")
			if logDir == "" {
				logDir = "/data/opsvault/logs"
			}
			_ = os.MkdirAll(logDir, 0755)
			logFile := filepath.Join(logDir, "opsvault.log")
			f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				// Redirect log output to the file instead of os.Stderr to prevent screen corruption
				log.SetOutput(f)
				defer func() {
					log.SetOutput(os.Stderr)
					f.Close()
				}()
			} else {
				// Fallback to discard standard output to prevent screen corruption
				log.SetOutput(io.Discard)
				defer log.SetOutput(os.Stderr)
			}

			// Set the logger listener to send messages to the Bubble Tea program
			logger.SetListener(func(logLine string) {
				program.Send(tui.LogLineMsg{Line: logLine})
			})
			defer logger.SetListener(nil)

			_, err = program.Run()
			return err
		},
	}
}
