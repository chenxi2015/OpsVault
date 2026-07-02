package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"OpsVault/internal/driver"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const refreshInterval = 5 * time.Second

type StatusProvider interface {
	Statuses() ([]driver.ServiceStatus, error)
}

type StaticStatusProvider map[string]driver.ServiceStatus

func (p StaticStatusProvider) Statuses() ([]driver.ServiceStatus, error) {
	names := make([]string, 0, len(p))
	for name := range p {
		names = append(names, name)
	}
	sort.Strings(names)
	services := make([]driver.ServiceStatus, 0, len(names))
	for _, name := range names {
		services = append(services, p[name])
	}
	return services, nil
}

type refreshTickMsg struct{}

type statusesLoadedMsg struct {
	services []driver.ServiceStatus
	err      error
}

type RootModel struct {
	active   int
	tabs     []string
	width    int
	height   int
	provider StatusProvider
	services []driver.ServiceStatus
	lastErr  error
}

func NewRootModel(provider ...StatusProvider) RootModel {
	var statusProvider StatusProvider = StaticStatusProvider{}
	if len(provider) > 0 && provider[0] != nil {
		statusProvider = provider[0]
	}
	return RootModel{
		tabs:     []string{"Dashboard", "Nginx", "Docker", "Config"},
		provider: statusProvider,
	}
}

func (m RootModel) Init() tea.Cmd {
	return tea.Batch(m.loadStatuses(), tickRefresh())
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
	case refreshTickMsg:
		return m, tea.Batch(m.loadStatuses(), tickRefresh())
	case statusesLoadedMsg:
		m.lastErr = msg.err
		if msg.err == nil {
			m.services = msg.services
		}
		return m, tickRefresh()
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
		body = DashboardView(m.services, m.lastErr)
	case 1:
		body = NginxPanelView(m.findService("nginx"), m.lastErr)
	case 2:
		body = DockerPanelView(filterDockerServices(m.services), m.lastErr)
	default:
		body = ConfigWizardView()
	}

	return fmt.Sprintf("%s\n\n%s\n\n%s",
		headerStyle.Render("OpsVault TUI"),
		lipgloss.JoinHorizontal(lipgloss.Top, tabs...),
		body,
	)
}

func (m RootModel) loadStatuses() tea.Cmd {
	return func() tea.Msg {
		services, err := m.provider.Statuses()
		return statusesLoadedMsg{services: services, err: err}
	}
}

func (m RootModel) findService(name string) *driver.ServiceStatus {
	for i := range m.services {
		if m.services[i].Name == name {
			service := m.services[i]
			return &service
		}
	}
	return nil
}

func filterDockerServices(services []driver.ServiceStatus) []driver.ServiceStatus {
	filtered := make([]driver.ServiceStatus, 0, len(services))
	for _, service := range services {
		if service.Mode == driver.ModeDocker {
			filtered = append(filtered, service)
		}
	}
	return filtered
}

func tickRefresh() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

func renderServiceLine(service driver.ServiceStatus) string {
	running := "down"
	if service.Running {
		running = "up"
	}
	parts := []string{
		service.Name,
		fmt.Sprintf("[%s]", running),
		service.Status,
	}
	if len(service.Ports) > 0 {
		parts = append(parts, "ports="+strings.Join(service.Ports, ","))
	}
	if service.Network != "" {
		parts = append(parts, "network="+service.Network)
	}
	if health := service.Details["health"]; health != "" && health != service.Status {
		parts = append(parts, "health="+health)
	}
	return "- " + strings.Join(parts, " ")
}
