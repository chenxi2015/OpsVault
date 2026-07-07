package cmd

import (
	"fmt"
	"strings"

	"OpsVault/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14")) // Cyan title

	instructionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")) // Dim gray instructions

	cursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("10")) // Green cursor >

	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("10")) // Green check [x]

	unselectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")) // Dim gray checkbox [ ]
)

type initItem struct {
	name     string
	selected bool
}

type initModel struct {
	items   []initItem
	cursor  int
	chosen  bool
	quitted bool
}

func newInitModel() initModel {
	return initModel{
		items: []initItem{
			{name: "nginx", selected: true},
			{name: "mysql", selected: true},
			{name: "redis", selected: true},
			{name: "rocketmq", selected: false},
			{name: "rabbitmq", selected: false},
			{name: "postgres", selected: false},
		},
	}
}

func (m initModel) Init() tea.Cmd {
	return nil
}

func (m initModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitted = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case " ", "space": // Accept both " " and "space" key messages
			m.items[m.cursor].selected = !m.items[m.cursor].selected
		case "enter":
			m.chosen = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m initModel) View() string {
	if m.quitted {
		return "Initialization cancelled.\n"
	}
	if m.chosen {
		return ""
	}

	var lines []string
	lines = append(lines, titleStyle.Render("OPSVAULT SERVICE INITIALIZATION"))
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("---------------------------------------------------------------"))
	lines = append(lines, "")

	for i, item := range m.items {
		cursorSymbol := "  "
		if m.cursor == i {
			cursorSymbol = cursorStyle.Render("> ")
		}

		checkbox := "[ ]"
		if item.selected {
			checkbox = selectedItemStyle.Render("[x]")
		} else {
			checkbox = unselectedItemStyle.Render("[ ]")
		}

		nameStr := item.name
		if m.cursor == i {
			nameStr = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render(item.name)
		} else {
			if item.selected {
				nameStr = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render(item.name)
			} else {
				nameStr = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(item.name)
			}
		}

		lines = append(lines, fmt.Sprintf("%s%s %s", cursorSymbol, checkbox, nameStr))
	}

	lines = append(lines, "")
	lines = append(lines, instructionStyle.Render("Press Up/Down (j/k) to navigate, Space to toggle."))
	lines = append(lines, instructionStyle.Render("Press Enter to confirm and begin installation."))
	lines = append(lines, instructionStyle.Render("Press Esc/q to quit."))

	borderBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(66).
		Render(strings.Join(lines, "\n"))

	return "\n" + borderBox + "\n"
}

func newInitCommand(cfg *viper.Viper, dockerFactory func() (*client.Client, error)) *cobra.Command {
	var (
		serviceNames string
		allServices  bool
		purge        bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize and install selected services",
		RunE: func(cmd *cobra.Command, args []string) error {
			provider := tui.NewRuntimeStatusProvider(cfg, dockerFactory)
			services := provider.Services()

			var targetServices []string
			if allServices {
				for _, svc := range services {
					targetServices = append(targetServices, svc.Name)
				}
			} else if serviceNames != "" {
				parts := strings.Split(serviceNames, ",")
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p != "" {
						targetServices = append(targetServices, p)
					}
				}
			} else {
				m := newInitModel()
				p := tea.NewProgram(&m)
				if _, err := p.Run(); err != nil {
					return err
				}
				if m.quitted {
					return nil
				}
				for _, item := range m.items {
					if item.selected {
						targetServices = append(targetServices, item.name)
					}
				}
			}

			if len(targetServices) == 0 {
				cmd.Println("No services selected for initialization.")
				return nil
			}

			cmd.Printf("Initializing selected services: %s\n", strings.Join(targetServices, ", "))

			for _, targetName := range targetServices {
				var foundSvc *tui.ServiceRef
				for i := range services {
					if services[i].Name == targetName {
						foundSvc = &services[i]
						break
					}
				}
				if foundSvc == nil {
					return fmt.Errorf("unknown service: %s", targetName)
				}

				cmd.Printf("\n--- [%s] ---\n", foundSvc.Name)
				if purge {
					cmd.Printf("Purging existing %s...\n", foundSvc.Name)
					if err := foundSvc.Driver.Uninstall(true); err != nil {
						cmd.Printf("Purge failed: %v (continuing)\n", err)
					}
				}

				cmd.Printf("Installing %s...\n", foundSvc.Name)
				if err := foundSvc.Driver.Install(); err != nil {
					return fmt.Errorf("failed to install %s: %w", foundSvc.Name, err)
				}

				cmd.Printf("Starting %s...\n", foundSvc.Name)
				if err := foundSvc.Driver.Start(); err != nil {
					return fmt.Errorf("failed to start %s: %w", foundSvc.Name, err)
				}

				cmd.Printf("%s initialized successfully.\n", foundSvc.Name)
			}

			cmd.Println("\nAll selected services initialized successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&serviceNames, "services", "", "Comma-separated list of services to initialize (e.g. nginx,mysql)")
	cmd.Flags().BoolVar(&allServices, "all", false, "Initialize all services")
	cmd.Flags().BoolVar(&purge, "purge", false, "Purge existing configurations/data before installation")

	return cmd
}
