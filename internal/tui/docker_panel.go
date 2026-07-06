package tui

import (
	"fmt"
	"strings"

	"OpsVault/internal/driver"

	"github.com/charmbracelet/lipgloss"
)

func DockerPanelView(m RootModel) string {
	// Filter registry for Docker services
	var dockerServices []ServiceRef
	for _, ref := range m.registry {
		if ref.Name != "nginx" {
			dockerServices = append(dockerServices, ref)
		}
	}

	if len(dockerServices) == 0 {
		return "No Docker services configured."
	}

	// Double check selected index bounds
	if m.selectedServiceIndex >= len(dockerServices) {
		return "Out of bounds"
	}
	selectedRef := dockerServices[m.selectedServiceIndex]

	// 1. Render Left Sidebar
	sidebarLines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Render("Docker Services"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("---------------"),
	}

	for idx, ref := range dockerServices {
		status := m.findService(ref.Name)
		runningState := "down"
		if status != nil && status.Running {
			runningState = "up"
		} else if status != nil && status.Status == "not installed" {
			runningState = "uninstalled"
		}

		// Color state indicator
		stateStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		if runningState == "up" {
			stateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
		} else if runningState == "uninstalled" {
			stateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
		}

		line := fmt.Sprintf("%-10s [%s]", ref.Name, stateStyle.Render(runningState))

		if m.focus == focusSidebar && idx == m.selectedServiceIndex {
			selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("10"))
			sidebarLines = append(sidebarLines, selectedStyle.Render("> "+line))
		} else {
			sidebarLines = append(sidebarLines, "  "+line)
		}
	}

	// 2. Render Right Detail Panel
	detailLines := []string{}
	status := m.findService(selectedRef.Name)

	var statusVal, imageVal, netVal, portsVal, pathVal, updateVal string
	statusVal = "not installed"
	imageVal = "-"
	netVal = "-"
	portsVal = "-"
	pathVal = "-"
	updateVal = "-"

	if m.config != nil {
		imageVal = m.config.GetString("docker.images." + selectedRef.Name)
		netVal = m.config.GetString("docker.network_name")
		pathVal = fmt.Sprintf("%s/%s", m.config.GetString("docker.data_root"), selectedRef.Name)
	}

	if status != nil {
		statusVal = status.Status
		if len(status.Ports) > 0 {
			portsVal = strings.Join(status.Ports, ", ")
		}
		if status.Network != "" {
			netVal = status.Network
		}
		if status.DataPath != "" {
			pathVal = status.DataPath
		}
		if img, ok := status.Details["image"]; ok {
			imageVal = img
		}
		updateVal = status.UpdatedAt.Format("15:04:05")
	}

	detailLines = append(detailLines,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Render(strings.ToUpper(selectedRef.Name)+" SERVICE DETAILS"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("--------------------------------------"),
		fmt.Sprintf("Status:       %s", formatStatus(statusVal)),
		fmt.Sprintf("Image Tag:    %s", imageVal),
		fmt.Sprintf("Network:      %s", netVal),
		fmt.Sprintf("Ports:        %s", portsVal),
		fmt.Sprintf("Data Path:    %s", pathVal),
		fmt.Sprintf("Last Update:  %s", updateVal),
		"",
	)

	// Available contextual Actions
	dummyStatus := driver.ServiceStatus{Name: selectedRef.Name, Status: statusVal}
	if status != nil {
		dummyStatus = *status
	}
	actions := AvailableServiceActions(dummyStatus, selectedRef)

	actionHints := []string{"Available Commands:"}
	for _, act := range actions {
		key := ""
		switch act.ID {
		case ActionStart:
			key = "s"
		case ActionStop:
			key = "x"
		case ActionRestart:
			key = "r"
		case ActionUpgrade:
			key = "u"
		case ActionInstall:
			key = "i"
		case ActionUninstall:
			key = "d"
		case ActionLogs:
			key = "l"
		}
		if key != "" {
			actionHints = append(actionHints, fmt.Sprintf("  [%s] %s", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render(key), act.Label))
		}
	}
	if len(actions) == 0 {
		actionHints = append(actionHints, "  (No actions available in current state)")
	}

	detailLines = append(detailLines, actionHints...)

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

func formatStatus(s string) string {
	if s == "running" || s == "healthy" {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render(s)
	}
	if s == "stopped" || s == "exited" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(s)
	}
	if s == "not installed" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(s)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(s)
}
