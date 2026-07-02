package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type RootModel struct {
	active int
	tabs   []string
	width  int
	height int
}

func NewRootModel() RootModel {
	return RootModel{
		tabs: []string{"Dashboard", "Nginx", "Docker", "Config"},
	}
}

func (m RootModel) Init() tea.Cmd {
	return nil
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "left", "h":
			if m.active > 0 {
				m.active--
			}
		case "right", "l":
			if m.active < len(m.tabs)-1 {
				m.active++
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m RootModel) View() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	tabs := make([]string, 0, len(m.tabs))
	for i, item := range m.tabs {
		if i == m.active {
			tabs = append(tabs, activeStyle.Render(item))
		} else {
			tabs = append(tabs, inactiveStyle.Render(item))
		}
	}

	body := ""
	switch m.active {
	case 0:
		body = DashboardView()
	case 1:
		body = NginxPanelView()
	case 2:
		body = DockerPanelView()
	default:
		body = ConfigWizardView()
	}

	return fmt.Sprintf("%s\n\n%s\n\n%s",
		headerStyle.Render("OpsVault TUI"),
		lipgloss.JoinHorizontal(lipgloss.Top, tabs...),
		body,
	)
}
