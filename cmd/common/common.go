package common

import (
	"fmt"
	"sort"
	"strings"

	"OpsVault/internal/driver"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	// Style definitions
	titleColor   = lipgloss.Color("86")  // Cyan/Teal
	labelColor   = lipgloss.Color("246") // Light Slate Gray
	valueColor   = lipgloss.Color("253") // Soft White
	borderColor  = lipgloss.Color("240") // Dark Slate Gray
	successColor = lipgloss.Color("10")  // Green
	warnColor    = lipgloss.Color("11")  // Yellow/Amber
	failColor    = lipgloss.Color("9")   // Red

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(titleColor)

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(labelColor)

	valueStyle = lipgloss.NewStyle().
			Foreground(valueColor)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 2)
)

func formatLabel(key string) string {
	switch strings.ToLower(key) {
	case "www_root":
		return "WWW Root"
	case "ssl_root":
		return "SSL Root"
	case "data_path":
		return "Data Path"
	case "pid":
		return "PID"
	case "image":
		return "Image"
	case "health":
		return "Health"
	case "error":
		return "Error"
	default:
		words := strings.Split(key, "_")
		for i, word := range words {
			if len(word) > 0 {
				words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
			}
		}
		return strings.Join(words, " ")
	}
}

func PrintStatus(cmd *cobra.Command, status *driver.ServiceStatus) {
	if status == nil {
		cmd.Println(lipgloss.NewStyle().Foreground(failColor).Render("No status available"))
		return
	}

	var content []string

	// Helper to add formatted row
	addRow := func(label, value string) {
		lbl := formatLabel(label) + ":"
		paddedLabel := fmt.Sprintf("%-14s", lbl)
		content = append(content, fmt.Sprintf("%s %s", labelStyle.Render(paddedLabel), valueStyle.Render(value)))
	}

	// 1. Mode
	addRow("mode", string(status.Mode))

	// 2. Status with colored dot
	var statusStr string
	if status.Running {
		dot := lipgloss.NewStyle().Foreground(successColor).Render("●")
		statusText := lipgloss.NewStyle().Foreground(successColor).Bold(true).Render(status.Status)
		statusStr = fmt.Sprintf("%s %s", dot, statusText)
	} else {
		dot := lipgloss.NewStyle().Foreground(failColor).Render("○")
		statusText := lipgloss.NewStyle().Foreground(failColor).Bold(true).Render(status.Status)
		statusStr = fmt.Sprintf("%s %s", dot, statusText)
	}
	addRow("status", statusStr)

	// 3. Running
	var runningStr string
	if status.Running {
		runningStr = lipgloss.NewStyle().Foreground(successColor).Render("true")
	} else {
		runningStr = lipgloss.NewStyle().Foreground(failColor).Render("false")
	}
	addRow("running", runningStr)

	// 4. Version
	if status.Version != "" {
		addRow("version", status.Version)
	}

	// 5. Ports
	if len(status.Ports) > 0 {
		addRow("ports", strings.Join(status.Ports, ", "))
	}

	// 6. Data Path
	if status.DataPath != "" {
		addRow("data_path", status.DataPath)
	}

	// 7. Network
	if status.Network != "" {
		addRow("network", status.Network)
	}

	// 8. PID
	if status.PID > 0 {
		addRow("pid", fmt.Sprintf("%d", status.PID))
	}

	// 9. Dynamic details
	if len(status.Details) > 0 {
		keys := make([]string, 0, len(status.Details))
		for key := range status.Details {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			val := status.Details[key]
			if key == "health" {
				if val == "healthy" {
					val = lipgloss.NewStyle().Foreground(successColor).Render(val)
				} else if val == "unhealthy" {
					val = lipgloss.NewStyle().Foreground(failColor).Render(val)
				} else if val != "" {
					val = lipgloss.NewStyle().Foreground(warnColor).Render(val)
				}
			}
			addRow(key, val)
		}
	}

	innerContent := strings.Join(content, "\n")

	// Calculate the maximum visible width of the lines to size the divider dynamically
	maxW := 0
	titleText := fmt.Sprintf(" %s STATUS ", strings.ToUpper(status.Name))
	titleLen := lipgloss.Width(titleText)
	if titleLen > maxW {
		maxW = titleLen
	}
	for _, line := range content {
		w := lipgloss.Width(line)
		if w > maxW {
			maxW = w
		}
	}
	// Fallback to a minimum width if content is very small
	if maxW < 30 {
		maxW = 30
	}

	styledTitle := titleStyle.Render(titleText)
	divider := lipgloss.NewStyle().Foreground(borderColor).Render(strings.Repeat("─", maxW))

	boxContent := fmt.Sprintf("%s\n%s\n%s", styledTitle, divider, innerContent)
	renderedBox := boxStyle.Render(boxContent)

	cmd.Println(renderedBox)
}

func RequireMode(actual driver.Mode, allowed ...driver.Mode) error {
	for _, item := range allowed {
		if actual == item {
			return nil
		}
	}
	values := make([]string, 0, len(allowed))
	for _, item := range allowed {
		values = append(values, string(item))
	}
	return fmt.Errorf("unsupported mode %q, allowed: %s", actual, strings.Join(values, ", "))
}
