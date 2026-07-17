package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ConfigCategory represents a grouped set of config keys.
type ConfigCategory struct {
	Name string
	Keys []string
}

// ConfigCategories defines all configuration groupings.
var ConfigCategories = []ConfigCategory{
	{
		Name: "Docker Global",
		Keys: []string{
			"docker.name_prefix",
			"docker.network_name",
			"docker.cidr",
			"docker.data_root",
			"docker.bind_ip",
		},
	},
	{
		Name: "Nginx",
		Keys: []string{
			"nginx.install_path",
			"nginx.www_root",
			"nginx.ssl_root",
			"nginx.wwwlogs_root",
			"nginx.source_root",
			"nginx.version",
			"nginx.pcre_version",
			"nginx.openssl_version",
			"nginx.run_user",
			"nginx.run_group",
		},
	},
	{
		Name: "MySQL",
		Keys: []string{
			"mysql.image",
			"mysql.port",
			"mysql.root_password",
		},
	},
	{
		Name: "Redis",
		Keys: []string{
			"redis.image",
			"redis.port",
			"redis.password",
		},
	},
	{
		Name: "RocketMQ",
		Keys: []string{
			"rocketmq.image",
			"rocketmq.namesrv_port",
			"rocketmq.broker_port",
		},
	},
	{
		Name: "RabbitMQ",
		Keys: []string{
			"rabbitmq.image",
			"rabbitmq.port",
			"rabbitmq.ui_port",
			"rabbitmq.admin_user",
			"rabbitmq.admin_pwd",
		},
	},
	{
		Name: "PostgreSQL",
		Keys: []string{
			"postgres.image",
			"postgres.port",
			"postgres.password",
		},
	},
	{
		Name: "MinIO",
		Keys: []string{
			"minio.image",
			"minio.port",
			"minio.console_port",
			"minio.root_user",
			"minio.root_password",
		},
	},
	{
		Name: "ELK Stack",
		Keys: []string{
			"elk.elasticsearch_image",
			"elk.elasticsearch_port",
			"elk.kibana_image",
			"elk.kibana_port",
			"elk.logstash_image",
			"elk.logstash_port",
			"elk.es_java_opts",
		},
	},
	{
		Name: "Jenkins",
		Keys: []string{
			"jenkins.image",
			"jenkins.port",
			"jenkins.agent_port",
		},
	},
	{
		Name: "GitLab",
		Keys: []string{
			"gitlab.image",
			"gitlab.port",
			"gitlab.ssh_port",
			"gitlab.https_port",
		},
	},
	{
		Name: "System Settings",
		Keys: []string{
			"log.level",
			"log.storage_path",
			"mode",
		},
	},
}

// ConfigKeys is kept for backward compatibility if needed, but categories are preferred.
var ConfigKeys = []string{
	"docker.name_prefix",
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
	"minio.image",
	"minio.port",
	"minio.console_port",
	"minio.root_user",
	"minio.root_password",
	"elk.elasticsearch_image",
	"elk.elasticsearch_port",
	"elk.kibana_image",
	"elk.kibana_port",
	"elk.logstash_image",
	"elk.logstash_port",
	"elk.es_java_opts",
}

// ConfigWizardView renders the config options list with categories on left and params on right.
func ConfigWizardView(m RootModel) string {
	// 1. Sidebar rendering
	sidebarLines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Render("Categories"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("------------"),
		"",
	}

	for idx, cat := range ConfigCategories {
		line := cat.Name
		if idx == m.selectedConfigCategory {
			if m.focus == focusSidebar {
				// Focused Category highlight
				selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("10"))
				sidebarLines = append(sidebarLines, selectedStyle.Render("> "+line))
			} else {
				// Unfocused but selected Category highlight
				selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
				sidebarLines = append(sidebarLines, selectedStyle.Render("* "+line))
			}
		} else {
			sidebarLines = append(sidebarLines, "  "+line)
		}
	}

	// 2. Details rendering
	detailLines := []string{}
	selectedCat := ConfigCategories[m.selectedConfigCategory]
	detailLines = append(detailLines,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Render(strings.ToUpper(selectedCat.Name)+" CONFIGURATIONS"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("--------------------------------------------"),
		"",
	)

	for idx, key := range selectedCat.Keys {
		val := ""
		if m.config != nil {
			val = m.config.GetString(key)
		}

		line := fmt.Sprintf("%-28s : %s", key, val)

		if idx == m.selectedConfigItem {
			if m.focus == focusDetail {
				// Focused Config Item highlight
				selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("10"))
				detailLines = append(detailLines, selectedStyle.Render("> "+line))
			} else {
				// Unfocused Config Item
				selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
				detailLines = append(detailLines, selectedStyle.Render("  "+line))
			}
		} else {
			detailLines = append(detailLines, "  "+line)
		}
	}

	// Pad detail lines to match height
	for len(detailLines) < 14 {
		detailLines = append(detailLines, "")
	}

	// Hints at the bottom
	hints := []string{
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Press [Tab] to switch sidebar/details pane."),
		lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Press [Enter] to select category / edit parameter."),
		lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Press [Esc] to exit details, [s] to save configurations."),
	}
	detailLines = append(detailLines, hints...)

	sidebarBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(24).
		Height(21).
		Render(strings.Join(sidebarLines, "\n"))

	detailBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(64).
		Height(21).
		Render(strings.Join(detailLines, "\n"))

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, "  ", detailBox)
}
