package tui

import (
	"strings"

	"OpsVault/internal/driver"
)

func NginxPanelView(service *driver.ServiceStatus, loadErr error) string {
	lines := []string{"Nginx panel"}
	if loadErr != nil {
		lines = append(lines, "", "status refresh error: "+loadErr.Error())
	}
	lines = append(lines, "")
	if service == nil {
		lines = append(lines, "Nginx status unavailable.")
	} else {
		lines = append(lines, renderServiceLine(*service))
		if service.DataPath != "" {
			lines = append(lines, "install_path="+service.DataPath)
		}
	}
	lines = append(lines, "", "- vhost add/del/list", "- ssl apply/renew/delete", "- binary status/log")
	return strings.Join(lines, "\n")
}
