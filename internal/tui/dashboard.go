package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func DashboardView(m RootModel) string {
	var lines []string
	lines = append(lines, "Services Dashboard Overview:")
	if m.lastErr != nil {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Last Error: "+m.lastErr.Error()))
	}
	lines = append(lines, "")

	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Create headers
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	lines = append(lines, fmt.Sprintf("  %-12s  %-12s  %-15s  %-20s",
		headerStyle.Render("Service"),
		headerStyle.Render("Status"),
		headerStyle.Render("Version"),
		headerStyle.Render("Ports")))
	lines = append(lines, "  "+borderStyle.Render(strings.Repeat("-", 62)))

	for idx, ref := range m.registry {
		status := m.findService(ref.Name)
		statusStr := "unknown"
		versionStr := "-"
		portsStr := "-"

		if status != nil {
			statusStr = status.Status
			if status.Version != "" {
				versionStr = status.Version
			}
			if len(status.Ports) > 0 {
				portsStr = strings.Join(status.Ports, ",")
			}
		}

		// Color status
		var statusColored string
		if statusStr == "running" || statusStr == "healthy" {
			statusColored = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(statusStr)
		} else if statusStr == "stopped" || statusStr == "exited" {
			statusColored = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(statusStr)
		} else if statusStr == "not installed" {
			statusColored = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(statusStr)
		} else {
			statusColored = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(statusStr)
		}

		// If nginx is not installed, it shows "not installed" but we can still show Nginx version from config
		if ref.Name == "nginx" && statusStr == "not installed" && m.config != nil {
			versionStr = m.config.GetString("nginx.version")
		}

		lineContent := fmt.Sprintf("%-12s  %-12s  %-15s  %-20s", ref.Name, statusColored, versionStr, portsStr)

		if m.focus == focusSidebar && idx == m.selectedServiceIndex {
			selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("10"))
			lines = append(lines, selectedStyle.Render("> "+lineContent))
		} else {
			lines = append(lines, "  "+lineContent)
		}
	}

	lines = append(lines, "")
	if m.focus == focusSidebar {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Press Up/Down to navigate, Enter to manage service, left/right arrow to switch tabs."))
	} else {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Press Tab to change focus, left/right arrow to switch tabs."))
	}

	// Action shortcuts hint
	lines = append(lines, "", "⚡ Quick Actions for selected service:")
	lines = append(lines, "  s: Start   x: Stop   r: Restart   l: Logs   d: Uninstall")

	return strings.Join(lines, "\n")
}
