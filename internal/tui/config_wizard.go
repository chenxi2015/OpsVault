package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ConfigKeys lists all editable configuration keys.
var ConfigKeys = []string{
	"docker.network_name",
	"docker.cidr",
	"docker.data_root",
	"mysql.image",
	"mysql.port",
	"mysql.root_password",
	"redis.image",
	"redis.port",
	"redis.password",
	"rocketmq.image",
	"rocketmq.namesrv_port",
	"rocketmq.broker_port",
	"rabbitmq.image",
	"rabbitmq.port",
	"rabbitmq.ui_port",
	"rabbitmq.admin_user",
	"rabbitmq.admin_pwd",
	"postgres.image",
	"postgres.port",
	"postgres.password",
}

// ConfigWizardView renders the config options list.
func ConfigWizardView(m RootModel) string {
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Render("OPSVAULT SYSTEM CONFIGURATIONS"))
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("-------------------------------------------------------------"))

	for idx, key := range ConfigKeys {
		val := ""
		if m.config != nil {
			val = m.config.GetString(key)
		}

		line := fmt.Sprintf("  %-25s : %s", key, val)

		// Highlight selected item
		if idx == m.selectedServiceIndex { // Reuse selectedServiceIndex for config items navigation
			selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("10"))
			lines = append(lines, selectedStyle.Render("> "+line[2:]))
		} else {
			lines = append(lines, line)
		}
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Press Up/Down to navigate, Enter to edit selected item."))
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Press [s] to save configuration changes to default.yaml."))

	borderBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(72).
		Height(14).
		Render(strings.Join(lines, "\n"))

	return borderBox
}
