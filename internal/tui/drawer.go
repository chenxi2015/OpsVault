package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *RootModel) renderDrawer() string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1)

	title := "Drawer (Tasks & Logs)"
	if m.drawerMode == drawerTasks {
		title = "📋 Execution Task Output"
	} else if m.drawerMode == drawerLogs {
		title = "📜 Recent Service Logs"
	}

	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Render(title)

	// Keep the output within bounds
	lines := strings.Split(m.drawerContent, "\n")
	maxLines := 10
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	content := strings.Join(lines, "\n")

	width := m.width - 4
	if width < 20 {
		width = 20
	}

	return borderStyle.Width(width).Render(
		fmt.Sprintf("%s\n\n%s", header, content),
	)
}
