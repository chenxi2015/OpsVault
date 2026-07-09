package tui

import (
	"fmt"
	"strings"

	"OpsVault/internal/system"

	"github.com/charmbracelet/lipgloss"
)

func DoctorPanelView(m RootModel) string {
	var lines []string
	lines = append(lines, "Environment Diagnostics (环境体检)")
	lines = append(lines, "")

	if m.doctorRunning && len(m.doctorItems) == 0 {
		lines = append(lines, "  ⏳ 正在收集运行环境信息，请稍候...", "")
		return strings.Join(lines, "\n")
	}

	if len(m.doctorItems) == 0 {
		lines = append(lines, "  无诊断结果，请按下 [r] 键运行环境体检。", "")
		return strings.Join(lines, "\n")
	}

	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))

	colNameStyle := lipgloss.NewStyle().Width(25)
	colStatusStyle := lipgloss.NewStyle().Width(10)

	lines = append(lines, fmt.Sprintf("  %s  %s  %s",
		colNameStyle.Render(headerStyle.Render("Check Item")),
		colStatusStyle.Render(headerStyle.Render("Status")),
		headerStyle.Render("Message")))
	lines = append(lines, "  "+borderStyle.Render(strings.Repeat("-", 75)))

	for _, item := range m.doctorItems {
		var statusColored string
		switch item.Status {
		case system.StatusOk:
			statusColored = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("[  OK  ]")
		case system.StatusWarn:
			statusColored = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11")).Render("[ WARN ]")
		case system.StatusFail:
			statusColored = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")).Render("[ FAIL ]")
		}

		// Handle multi-line messages (like tables) to align properly under the Message column
		msgLines := strings.Split(item.Message, "\n")
		lineContent := fmt.Sprintf("  %s  %s  %s",
			colNameStyle.Render(item.Name),
			colStatusStyle.Render(statusColored),
			msgLines[0])
		lines = append(lines, lineContent)

		if len(msgLines) > 1 {
			// If it's a table or multi-line block, use a smaller indent (e.g., 4 spaces)
			// to prevent content wrapping and misalignment in terminal.
			for i := 1; i < len(msgLines); i++ {
				lines = append(lines, "    "+msgLines[i])
			}
		}

		if item.Suggestion != "" && item.Status != system.StatusOk {
			suggestionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
			lines = append(lines, suggestionStyle.Render(fmt.Sprintf("    --> Suggestion: %s", item.Suggestion)))
		}
	}

	lines = append(lines, "")
	if m.doctorRunning {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("  ⏳ 正在刷新环境信息..."))
	} else {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("  Press [r] to re-run diagnostics. left/right arrow to switch tabs."))
	}

	return strings.Join(lines, "\n")
}
