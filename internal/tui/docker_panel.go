package tui

import (
	"strings"

	"OpsVault/internal/driver"
)

func DockerPanelView(services []driver.ServiceStatus, loadErr error) string {
	lines := []string{"Docker panel"}
	if loadErr != nil {
		lines = append(lines, "", "status refresh error: "+loadErr.Error())
	}
	lines = append(lines, "")
	if len(services) == 0 {
		lines = append(lines, "No Docker services loaded.")
	} else {
		for _, service := range services {
			lines = append(lines, renderServiceLine(service))
		}
	}
	lines = append(lines, "", "- install/start/stop/restart/uninstall/upgrade", "- logs and status for all middleware containers")
	return strings.Join(lines, "\n")
}
