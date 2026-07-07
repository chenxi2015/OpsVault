package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"OpsVault/internal/driver"
	"OpsVault/internal/system"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/client"
	"github.com/spf13/viper"
)

const refreshInterval = 5 * time.Second

type focusRegion int

const (
	focusSidebar focusRegion = iota
	focusDetail
	focusDrawer
)

type drawerMode int

const (
	drawerHidden drawerMode = iota
	drawerTasks
	drawerLogs
)

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

type doctorFinishedMsg struct {
	items []system.DiagnosticItem
	err   error
}

type RootModel struct {
	active               int
	tabs                 []string
	width                int
	height               int
	provider             StatusProvider
	services             []driver.ServiceStatus
	lastErr              error
	focus                focusRegion
	drawerMode           drawerMode
	drawerContent        string
	registry             []ServiceRef
	dockerClient         *client.Client
	config               *viper.Viper
	selectedServiceIndex int

	// Nginx specific states
	selectedNginxSubMode int // 0: Service, 1: VHosts, 2: Certificates
	selectedVHostIndex   int
	selectedCertIndex    int
	nginxVHosts          []map[string]string

	// Confirmation flow
	confirming      bool
	confirmPrompt   string
	confirmCallback func() tea.Cmd

	// Editing input form
	editing         bool
	textInputPrompt string
	textInputState  string // "vhost_domain", "vhost_root", "ssl_apply_domain"
	textInputValue  string

	// Doctor specific states
	doctorItems   []system.DiagnosticItem
	doctorRunning bool
}

func NewRootModel(provider ...StatusProvider) RootModel {
	var statusProvider StatusProvider = StaticStatusProvider{}
	if len(provider) > 0 && provider[0] != nil {
		statusProvider = provider[0]
	}

	m := RootModel{
		tabs:  []string{"Dashboard", "Nginx", "Docker", "Doctor", "Config"},
		focus: focusSidebar,
	}

	if rt, ok := statusProvider.(RuntimeStatusProvider); ok {
		m.provider = rt
		m.config = rt.config
		m.registry = rt.Services()
		m.dockerClient, _ = rt.dockerFactory()
	} else {
		m.provider = statusProvider
		m.registry = []ServiceRef{
			{Name: "nginx"},
			{Name: "mysql"},
			{Name: "redis"},
			{Name: "rocketmq"},
			{Name: "rabbitmq"},
			{Name: "postgres"},
		}
	}

	return m
}

func (m *RootModel) Init() tea.Cmd {
	return tea.Batch(m.loadStatuses(), tickRefresh())
}

func (m *RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle input text forms if editing
		if m.editing {
			switch msg.String() {
			case "enter":
				return m.handleInputSubmit()
			case "backspace":
				if len(m.textInputValue) > 0 {
					m.textInputValue = m.textInputValue[:len(m.textInputValue)-1]
				}
			case "esc":
				m.editing = false
				m.textInputValue = ""
			default:
				if len(msg.Runes) > 0 {
					m.textInputValue += string(msg.Runes)
				}
			}
			return m, nil
		}

		// Handle confirmation prompt
		if m.confirming {
			switch msg.String() {
			case "y", "enter":
				m.confirming = false
				if m.confirmCallback != nil {
					cmd := m.confirmCallback()
					m.confirmCallback = nil
					// Automatically show task drawer
					m.drawerMode = drawerTasks
					m.drawerContent = "Executing dangerous action..."
					return m, cmd
				}
			case "n", "esc":
				m.confirming = false
				m.confirmCallback = nil
			}
			return m, nil
		}

		// Main navigation and hotkeys
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.cycleFocus()
		case "left", "h":
			if m.active > 0 {
				m.active--
				m.resetSubNavigation()
				if m.active == 3 && len(m.doctorItems) == 0 {
					m.doctorRunning = true
					return m, m.runDiagnosticsCmd()
				}
			}
		case "right", "l":
			if m.active < len(m.tabs)-1 {
				m.active++
				m.resetSubNavigation()
				if m.active == 3 && len(m.doctorItems) == 0 {
					m.doctorRunning = true
					return m, m.runDiagnosticsCmd()
				}
			}
		case "up", "k":
			m.moveSelection(-1)
		case "down", "j":
			m.moveSelection(1)
		case "esc":
			m.drawerMode = drawerHidden
		}

		// Tab specific shortcuts and commands
		return m.handleShortcuts(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case refreshTickMsg:
		return m, m.loadStatuses() // Load statuses directly without queueing double ticks

	case statusesLoadedMsg:
		m.lastErr = msg.err
		if msg.err == nil {
			m.services = msg.services
			// Sync Nginx vhosts if Nginx driver is loaded
			if nginxSvc := m.findRegistry("nginx"); nginxSvc != nil {
				if vh, ok := nginxSvc.Driver.(driver.VHostManager); ok {
					if list, err := vh.ListVHosts(); err == nil {
						m.nginxVHosts = list
					}
				}
			}
		}
		return m, tickRefresh() // Start next tick refresh interval

	case doctorFinishedMsg:
		m.doctorRunning = false
		if msg.err != nil {
			m.lastErr = msg.err
		} else {
			m.doctorItems = msg.items
		}
		return m, nil

	case taskFinishedMsg:
		m.drawerContent = msg.Output
		if msg.Err != nil {
			m.lastErr = msg.Err
		}
		// Refresh statuses immediately after a task finishes
		return m, m.loadStatuses()
	}
	return m, nil
}

func (m *RootModel) View() string {
	// Brand and Title Styling
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Background(lipgloss.Color("235")).
		Padding(0, 2).
		MarginBottom(1)

	// Tab Bar styling
	activeTabStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("10")).
		Background(lipgloss.Color("236")).
		Padding(0, 2)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Padding(0, 2)

	tabs := make([]string, 0, len(m.tabs))
	for i, item := range m.tabs {
		if i == m.active {
			tabs = append(tabs, activeTabStyle.Render(item))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(item))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	// Main Panel body view delegation
	body := ""
	switch m.active {
	case 0:
		body = m.dashboardView()
	case 1:
		body = m.nginxView()
	case 2:
		body = m.dockerView()
	case 3:
		body = m.doctorView()
	default:
		body = ConfigWizardView(*m)
	}

	// Bottom task / log drawer
	drawer := ""
	if m.drawerMode != drawerHidden {
		drawer = m.renderDrawer()
	}

	// Simple Dialog overlay for editing / confirmation
	overlay := ""
	if m.editing {
		overlay = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("208")).
			Padding(1, 2).
			Render(fmt.Sprintf("%s\n\n> %s█", m.textInputPrompt, m.textInputValue))
	} else if m.confirming {
		overlay = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("196")).
			Bold(true).
			Padding(1, 2).
			Render(fmt.Sprintf("%s\n\n[y/Enter] Confirm   [n/Esc] Cancel", m.confirmPrompt))
	}

	fullLayout := fmt.Sprintf("%s\n\n%s\n\n%s",
		headerStyle.Render("OpsVault Operations Console"),
		tabBar,
		body,
	)

	if overlay != "" {
		fullLayout = fmt.Sprintf("%s\n\n%s", fullLayout, overlay)
	}

	if drawer != "" {
		fullLayout = fmt.Sprintf("%s\n\n%s", fullLayout, drawer)
	}

	return fullLayout
}

func (m *RootModel) loadStatuses() tea.Cmd {
	return func() tea.Msg {
		services, err := m.provider.Statuses()
		return statusesLoadedMsg{services: services, err: err}
	}
}

func (m *RootModel) findService(name string) *driver.ServiceStatus {
	for i := range m.services {
		if m.services[i].Name == name {
			service := m.services[i]
			return &service
		}
	}
	return nil
}

func (m *RootModel) findRegistry(name string) *ServiceRef {
	for i := range m.registry {
		if m.registry[i].Name == name {
			return &m.registry[i]
		}
	}
	return nil
}

func (m *RootModel) cycleFocus() {
	if m.drawerMode == drawerHidden {
		if m.focus == focusSidebar {
			m.focus = focusDetail
		} else {
			m.focus = focusSidebar
		}
	} else {
		switch m.focus {
		case focusSidebar:
			m.focus = focusDetail
		case focusDetail:
			m.focus = focusDrawer
		case focusDrawer:
			m.focus = focusSidebar
		}
	}
}

func (m *RootModel) resetSubNavigation() {
	m.focus = focusSidebar
	m.selectedServiceIndex = 0
	m.selectedNginxSubMode = 0
	m.selectedVHostIndex = 0
	m.selectedCertIndex = 0
	m.confirming = false
	m.editing = false
}

func (m *RootModel) handleInputSubmit() (tea.Model, tea.Cmd) {
	m.editing = false
	val := m.textInputValue
	m.textInputValue = ""

	if strings.HasPrefix(m.textInputState, "config|") {
		configKey := strings.TrimPrefix(m.textInputState, "config|")
		if m.config != nil {
			if strings.Contains(configKey, "port") {
				if intVal, err := strconv.Atoi(val); err == nil {
					m.config.Set(configKey, intVal)
				} else {
					m.config.Set(configKey, val)
				}
			} else {
				m.config.Set(configKey, val)
			}
		}
		return m, nil
	}

	switch m.textInputState {
	case "vhost_domain":
		if val == "" {
			return m, nil
		}
		m.textInputState = "vhost_root|" + val
		m.textInputPrompt = fmt.Sprintf("Enter Virtual Host Root Directory (Domain: %s):", val)
		m.textInputValue = "/data/wwwroot/" + val
		m.editing = true
		return m, nil

	default:
		if strings.HasPrefix(m.textInputState, "vhost_root|") {
			domain := strings.TrimPrefix(m.textInputState, "vhost_root|")
			root := val
			if root == "" {
				root = "/data/wwwroot/" + domain
			}
			svc := m.findRegistry("nginx")
			if svc == nil {
				return m, nil
			}
			m.drawerMode = drawerTasks
			m.drawerContent = fmt.Sprintf("Adding Virtual Host for %s...", domain)
			return m, runAction(m.config, m.dockerClient, *svc, Action{ID: "vhost_add"}, map[string]string{
				"domain": domain,
				"root":   root,
			})
		}

		if m.textInputState == "ssl_apply_domain" {
			if val == "" {
				return m, nil
			}
			svc := m.findRegistry("nginx")
			if svc == nil {
				return m, nil
			}
			m.drawerMode = drawerTasks
			m.drawerContent = fmt.Sprintf("Applying SSL certificate for %s...", val)
			return m, runAction(m.config, m.dockerClient, *svc, Action{ID: "ssl_apply"}, map[string]string{
				"domain": val,
			})
		}
	}
	return m, nil
}

func (m *RootModel) handleShortcuts(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.active == 3 { // Doctor Tab
		if msg.String() == "r" {
			m.doctorRunning = true
			return m, m.runDiagnosticsCmd()
		}
		return m, nil
	}

	if m.active == 4 {
		if msg.String() == "enter" && m.selectedServiceIndex < len(ConfigKeys) {
			configKey := ConfigKeys[m.selectedServiceIndex]
			m.editing = true
			m.textInputPrompt = fmt.Sprintf("Enter new value for %s:", configKey)
			m.textInputState = "config|" + configKey
			m.textInputValue = m.config.GetString(configKey)
			return m, nil
		}
		if msg.String() == "s" {
			m.drawerMode = drawerTasks
			m.drawerContent = "Saving configurations to configs/default.yaml..."
			return m, func() tea.Msg {
				var err error
				if m.config != nil {
					if m.config.ConfigFileUsed() != "" {
						err = m.config.WriteConfig()
					} else {
						var targetPath string
						if exePath, errExe := os.Executable(); errExe == nil {
							targetPath = filepath.Join(filepath.Dir(exePath), "configs", "default.yaml")
						} else {
							targetPath = filepath.Join("configs", "default.yaml")
						}
						if errDir := os.MkdirAll(filepath.Dir(targetPath), 0755); errDir == nil {
							err = m.config.WriteConfigAs(targetPath)
							if err == nil {
								m.config.SetConfigFile(targetPath)
							}
						} else {
							err = errDir
						}
					}
				}
				output := "Configuration successfully saved."
				if err != nil {
					output = fmt.Sprintf("Error saving configuration: %v", err)
				}
				return taskFinishedMsg{
					ServiceName: "config",
					ActionName:  "save",
					Output:      output,
					Err:         err,
				}
			}
		}
		return m, nil
	}

	var selectedSvc ServiceRef
	var hasSvc bool

	switch m.active {
	case 2:
		dockServices := filterDockerServices(m.services)
		if len(dockServices) > 0 && m.selectedServiceIndex < len(dockServices) {
			targetName := dockServices[m.selectedServiceIndex].Name
			if reg := m.findRegistry(targetName); reg != nil {
				selectedSvc = *reg
				hasSvc = true
			}
		}
	case 1:
		if m.selectedNginxSubMode == 0 {
			if reg := m.findRegistry("nginx"); reg != nil {
				selectedSvc = *reg
				hasSvc = true
			}
		}
	case 0:
		if len(m.registry) > 0 && m.selectedServiceIndex < len(m.registry) {
			selectedSvc = m.registry[m.selectedServiceIndex]
			hasSvc = true
		}
	}

	if !hasSvc {
		// Handle Nginx VHosts and Certificates sub-mode special shortcuts
		if m.active == 1 {
			if m.selectedNginxSubMode == 1 { // VHosts Mode
				switch msg.String() {
				case "a": // Add vhost
					m.editing = true
					m.textInputPrompt = "Enter Virtual Host Domain Name:"
					m.textInputState = "vhost_domain"
					m.textInputValue = ""
					return m, nil
				case "d": // Delete vhost
					if len(m.nginxVHosts) > 0 && m.selectedVHostIndex < len(m.nginxVHosts) {
						domain := m.nginxVHosts[m.selectedVHostIndex]["domain"]
						m.confirming = true
						m.confirmPrompt = fmt.Sprintf("DANGER: Delete VHost for %s (websites files will not be deleted)?", domain)
						m.confirmCallback = func() tea.Cmd {
							svc := m.findRegistry("nginx")
							return runAction(m.config, m.dockerClient, *svc, Action{ID: "vhost_del"}, map[string]string{
								"domain":      domain,
								"delete-root": "false",
							})
						}
						return m, nil
					}
				}
			} else if m.selectedNginxSubMode == 2 { // Certificates Mode
				switch msg.String() {
				case "a": // Apply SSL
					m.editing = true
					m.textInputPrompt = "Enter Domain Name for Let's Encrypt SSL:"
					m.textInputState = "ssl_apply_domain"
					m.textInputValue = ""
					return m, nil
				case "r": // Renew SSL
					if len(m.nginxVHosts) > 0 && m.selectedCertIndex < len(m.nginxVHosts) {
						domain := m.nginxVHosts[m.selectedCertIndex]["domain"]
						m.drawerMode = drawerTasks
						m.drawerContent = fmt.Sprintf("Renewing SSL for %s...", domain)
						svc := m.findRegistry("nginx")
						return m, runAction(m.config, m.dockerClient, *svc, Action{ID: "ssl_renew"}, map[string]string{
							"domain": domain,
						})
					}
				case "d": // Delete SSL
					if len(m.nginxVHosts) > 0 && m.selectedCertIndex < len(m.nginxVHosts) {
						domain := m.nginxVHosts[m.selectedCertIndex]["domain"]
						m.confirming = true
						m.confirmPrompt = fmt.Sprintf("DANGER: Delete SSL Certificate for %s?", domain)
						m.confirmCallback = func() tea.Cmd {
							svc := m.findRegistry("nginx")
							return runAction(m.config, m.dockerClient, *svc, Action{ID: "ssl_delete"}, map[string]string{
								"domain": domain,
							})
						}
						return m, nil
					}
				}
			}
		}
		return m, nil
	}

	var actionID ActionID
	switch msg.String() {
	case "s":
		actionID = ActionStart
	case "x":
		actionID = ActionStop
	case "r":
		actionID = ActionRestart
	case "u":
		actionID = ActionUpgrade
	case "i":
		actionID = ActionInstall
	case "l":
		actionID = ActionLogs
	case "d":
		actionID = ActionUninstall
	case "v":
		if selectedSvc.Name == "rocketmq" {
			m.drawerMode = drawerTasks
			m.drawerContent = "Querying RocketMQ broker version..."
			return m, runAction(m.config, m.dockerClient, selectedSvc, Action{ID: ActionVersion, Label: "Version Query"}, nil)
		}
	case "q":
		if selectedSvc.Name == "rocketmq" {
			m.drawerMode = drawerTasks
			m.drawerContent = "Querying RocketMQ DLQ stats..."
			return m, runAction(m.config, m.dockerClient, selectedSvc, Action{ID: ActionDLQStat, Label: "DLQ Stats"}, nil)
		}
	case "t":
		m.drawerMode = drawerTasks
		return m, nil
	}

	if actionID == "" {
		if msg.String() == "enter" && m.active == 0 {
			if selectedSvc.Name == "nginx" {
				m.active = 1
			} else {
				m.active = 2
				dockServices := filterDockerServices(m.services)
				for idx, ds := range dockServices {
					if ds.Name == selectedSvc.Name {
						m.selectedServiceIndex = idx
						break
					}
				}
			}
			m.focus = focusSidebar
			return m, nil
		}
		return m, nil
	}

	statusPtr := m.findService(selectedSvc.Name)
	if statusPtr == nil {
		statusPtr = &driver.ServiceStatus{Name: selectedSvc.Name, Status: "not installed"}
	}
	status := *statusPtr

	actions := AvailableServiceActions(status, selectedSvc)
	var targetAction *Action
	for i := range actions {
		if actions[i].ID == actionID {
			targetAction = &actions[i]
			break
		}
	}

	if targetAction == nil {
		return m, nil
	}

	if targetAction.ID == ActionLogs {
		m.drawerMode = drawerLogs
		m.drawerContent = "Fetching logs..."
		return m, runAction(m.config, m.dockerClient, selectedSvc, *targetAction, nil)
	}

	if targetAction.Dangerous {
		m.confirming = true
		m.confirmPrompt = fmt.Sprintf("DANGER: Are you sure you want to %s %s?", targetAction.Label, selectedSvc.Name)
		m.confirmCallback = func() tea.Cmd {
			params := map[string]string{"purge": "false"}
			return runAction(m.config, m.dockerClient, selectedSvc, *targetAction, params)
		}
		return m, nil
	}

	m.drawerMode = drawerTasks
	m.drawerContent = fmt.Sprintf("Running %s on %s...", targetAction.Label, selectedSvc.Name)
	return m, runAction(m.config, m.dockerClient, selectedSvc, *targetAction, nil)
}

func (m *RootModel) dashboardView() string {
	return DashboardView(*m)
}

func (m *RootModel) dockerView() string {
	return DockerPanelView(*m)
}

func (m *RootModel) nginxView() string {
	return NginxPanelView(*m)
}

func (m *RootModel) doctorView() string {
	return DoctorPanelView(*m)
}

func (m *RootModel) runDiagnosticsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		items, err := system.RunDiagnostics(ctx, m.config, m.dockerClient)
		return doctorFinishedMsg{items: items, err: err}
	}
}

func tickRefresh() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg {
		return refreshTickMsg{}
	})
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

func (m *RootModel) moveSelection(dir int) {
	if m.focus == focusSidebar {
		switch m.active {
		case 0, 2:
			listLen := len(m.registry) - 1 // Excluding nginx
			if m.active == 0 {
				listLen = len(m.registry)
			}
			if m.active == 2 {
				listLen = len(filterDockerServices(m.services))
				if listLen == 0 {
					listLen = 5
				}
			}
			m.selectedServiceIndex += dir
			if m.selectedServiceIndex < 0 {
				m.selectedServiceIndex = listLen - 1
			} else if m.selectedServiceIndex >= listLen {
				m.selectedServiceIndex = 0
			}
		case 1:
			m.selectedNginxSubMode += dir
			if m.selectedNginxSubMode < 0 {
				m.selectedNginxSubMode = 2
			} else if m.selectedNginxSubMode > 2 {
				m.selectedNginxSubMode = 0
			}
		case 4:
			listLen := len(ConfigKeys)
			m.selectedServiceIndex += dir
			if m.selectedServiceIndex < 0 {
				m.selectedServiceIndex = listLen - 1
			} else if m.selectedServiceIndex >= listLen {
				m.selectedServiceIndex = 0
			}
		}
	} else if m.focus == focusDetail {
		if m.active == 1 {
			if m.selectedNginxSubMode == 1 && len(m.nginxVHosts) > 0 {
				m.selectedVHostIndex += dir
				if m.selectedVHostIndex < 0 {
					m.selectedVHostIndex = len(m.nginxVHosts) - 1
				} else if m.selectedVHostIndex >= len(m.nginxVHosts) {
					m.selectedVHostIndex = 0
				}
			} else if m.selectedNginxSubMode == 2 && len(m.nginxVHosts) > 0 {
				m.selectedCertIndex += dir
				if m.selectedCertIndex < 0 {
					m.selectedCertIndex = len(m.nginxVHosts) - 1
				} else if m.selectedCertIndex >= len(m.nginxVHosts) {
					m.selectedCertIndex = 0
				}
			}
		}
	}
}
