package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func NginxPanelView(m RootModel) string {
	// 1. Render Left Sidebar (Sub-modes selection)
	sidebarLines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Render("Nginx Resources"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("---------------"),
	}

	subModes := []string{"Service", "VHosts", "Certificates"}
	for idx, mode := range subModes {
		if m.focus == focusSidebar && idx == m.selectedNginxSubMode {
			selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("10"))
			sidebarLines = append(sidebarLines, selectedStyle.Render("> "+mode))
		} else {
			sidebarLines = append(sidebarLines, "  "+mode)
		}
	}

	// 2. Render Right Detail Panel based on sub-mode
	detailLines := []string{}
	selectedMode := subModes[m.selectedNginxSubMode]

	switch selectedMode {
	case "Service":
		status := m.findService("nginx")
		var statusVal, versionVal, pathVal, wwwRootVal, sslRootVal string
		statusVal = "not installed"
		versionVal = "-"
		pathVal = "-"
		wwwRootVal = "-"
		sslRootVal = "-"
		pidVal := 0

		if m.config != nil {
			versionVal = m.config.GetString("nginx.version")
			pathVal = m.config.GetString("nginx.install_path")
			wwwRootVal = m.config.GetString("nginx.www_root")
			sslRootVal = m.config.GetString("nginx.ssl_root")
		}

		if status != nil {
			statusVal = status.Status
			pidVal = status.PID
			if status.Version != "" {
				versionVal = status.Version
			}
			if status.DataPath != "" {
				pathVal = status.DataPath
			}
			if r, ok := status.Details["www_root"]; ok {
				wwwRootVal = r
			}
			if s, ok := status.Details["ssl_root"]; ok {
				sslRootVal = s
			}
		}

		detailLines = append(detailLines,
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Render("NGINX BINARY SERVICE"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("--------------------------------------"),
			fmt.Sprintf("Status:       %s", formatStatus(statusVal)),
			fmt.Sprintf("PID:          %d", pidVal),
			fmt.Sprintf("Version:      %s", versionVal),
			fmt.Sprintf("Install Path: %s", pathVal),
			fmt.Sprintf("Www Root:     %s", wwwRootVal),
			fmt.Sprintf("SSL Root:     %s", sslRootVal),
			"",
			"Available Commands:",
			fmt.Sprintf("  [%s] Start   [%s] Stop   [%s] Restart   [%s] Reload",
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("s"),
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("x"),
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("r"),
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("c"),
			),
			fmt.Sprintf("  [%s] Logs    [%s] Uninstall  [%s] Upgrade",
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("l"),
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("d"),
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("u"),
			),
		)

	case "VHosts":
		detailLines = append(detailLines,
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Render("VIRTUAL HOSTS (VHOSTS)"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("--------------------------------------"),
		)

		var selectedPath string
		if len(m.nginxVHosts) == 0 {
			detailLines = append(detailLines, "No Virtual Hosts configured.", "")
		} else {
			for idx, vh := range m.nginxVHosts {
				domain := strings.TrimSuffix(vh["domain"], ".conf")
				prefix := "  "
				rowStyle := lipgloss.NewStyle()
				if m.focus == focusDetail && idx == m.selectedVHostIndex {
					prefix = "> "
					rowStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
					selectedPath = vh["path"]
				}
				detailLines = append(detailLines, prefix+rowStyle.Render(domain))
			}
			detailLines = append(detailLines, "")
		}

		if selectedPath != "" {
			pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
			detailLines = append(detailLines, pathStyle.Render(fmt.Sprintf("Config: %s", selectedPath)), "")
		}

		detailLines = append(detailLines,
			"Commands:",
			fmt.Sprintf("  [%s] Add Virtual Host   [%s] Delete Selected",
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("a"),
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("d"),
			),
		)

	case "Certificates":
		detailLines = append(detailLines,
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Render("SSL CERTIFICATES (LET'S ENCRYPT)"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("--------------------------------------"),
		)

		if len(m.nginxVHosts) == 0 {
			detailLines = append(detailLines, "No domains configured to manage SSL certificates.", "")
		} else {
			sslRoot := "/data/ssl"
			if m.config != nil {
				sslRoot = m.config.GetString("nginx.ssl_root")
			}

			var selectedCertPath string
			for idx, vh := range m.nginxVHosts {
				domain := strings.TrimSuffix(vh["domain"], ".conf")
				certPath := filepath.Join(sslRoot, domain, "fullchain.pem")

				sslStatus := "HTTP Only"
				sslStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
				if _, err := os.Stat(certPath); err == nil {
					sslStatus = "HTTPS Enabled"
					sslStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
				}

				prefix := "  "
				rowStyle := lipgloss.NewStyle()
				if m.focus == focusDetail && idx == m.selectedCertIndex {
					prefix = "> "
					rowStyle = lipgloss.NewStyle().Bold(true)
					selectedCertPath = certPath
				}
				detailLines = append(detailLines, fmt.Sprintf("%s%s (%s)", prefix, rowStyle.Render(domain), sslStyle.Render(sslStatus)))
			}
			detailLines = append(detailLines, "")

			if selectedCertPath != "" {
				pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
				detailLines = append(detailLines, pathStyle.Render(fmt.Sprintf("Cert: %s", selectedCertPath)), "")
			}
		}

		detailLines = append(detailLines,
			"Commands:",
			fmt.Sprintf("  [%s] Apply SSL (Let's Encrypt)  [%s] Renew Selected  [%s] Delete SSL",
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("a"),
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("r"),
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("d"),
			),
		)
	}

	// Joint Layout using columns
	sidebarBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(26).
		Height(12).
		Render(strings.Join(sidebarLines, "\n"))

	detailBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(48).
		Height(12).
		Render(strings.Join(detailLines, "\n"))

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, "  ", detailBox)
}
