package tui

import (
	"strings"

	"OpsVault/internal/driver"
)

func DashboardView(services []driver.ServiceStatus, loadErr error) string {
	lines := []string{"Services overview"}
	if loadErr != nil {
		lines = append(lines, "", "status refresh error: "+loadErr.Error())
	}
	if len(services) == 0 {
		lines = append(lines, "", "No services loaded yet.")
	} else {
		lines = append(lines, "")
		for _, service := range services {
			lines = append(lines, renderServiceLine(service))
		}
	}
	lines = append(lines, "", "Use ←/→ to switch panels, q to quit.")
	return strings.Join(lines, "\n")
}
