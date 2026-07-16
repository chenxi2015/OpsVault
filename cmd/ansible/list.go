package ansiblecmd

import (
	"fmt"

	"OpsVault/internal/driver/ansible"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

func (c *commandSet) newListCommand() *cobra.Command {
	var group string
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"hosts", "inventory"},
		Short:   "List configured host groups and servers from inventory",
		Long:    `Quickly view all configured Ansible host groups, server IPs, ports, login users, and authentication methods without inspecting the configuration file manually.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := ansible.LoadConfig(c.config)
			if err != nil {
				return fmt.Errorf("failed to load ansible config: %w", err)
			}

			if len(cfg.Groups) == 0 {
				cmd.Println("No host groups configured in inventory.")
				return nil
			}

			var targetGroups []ansible.GroupConfig
			if group != "all" && group != "" {
				for _, g := range cfg.Groups {
					if g.Name == group {
						targetGroups = append(targetGroups, g)
					}
				}
				if len(targetGroups) == 0 {
					return fmt.Errorf("group not found in inventory configuration: %s", group)
				}
				cmd.Printf("Listing servers in host group: %s\n", group)
			} else {
				targetGroups = cfg.Groups
				cmd.Println("Listing all configured host groups and servers:")
			}

			// Render Table Headers
			groupHeader := headerStyle.Width(16).Render("GROUP")
			ipHeader := headerStyle.Width(18).Render("HOST IP")
			portHeader := headerStyle.Width(8).Render("PORT")
			userHeader := headerStyle.Width(12).Render("USER")
			authHeader := headerStyle.Width(38).Render("AUTH METHOD")
			pyHeader := headerStyle.Width(22).Render("PYTHON INTERPRETER")

			cmd.Println("\n" + lipgloss.JoinHorizontal(lipgloss.Top, groupHeader, ipHeader, portHeader, userHeader, authHeader, pyHeader))

			totalGroups := len(targetGroups)
			totalHosts := 0

			for _, g := range targetGroups {
				groupName := g.Name
				if groupName == "" {
					groupName = "default"
				}

				if len(g.Hosts) == 0 {
					gRow := rowStyle.Width(16).Render(groupName)
					ipRow := rowStyle.Width(18).Render("N/A (No hosts)")
					portRow := rowStyle.Width(8).Render("-")
					userRow := rowStyle.Width(12).Render("-")
					authRow := rowStyle.Width(38).Render("-")
					pyRow := rowStyle.Width(22).Render("-")
					cmd.Println(lipgloss.JoinHorizontal(lipgloss.Top, gRow, ipRow, portRow, userRow, authRow, pyRow))
					continue
				}

				for _, h := range g.Hosts {
					if h.IP == "" {
						continue
					}
					totalHosts++

					gRow := rowStyle.Width(16).Render(groupName)
					ipRow := rowStyle.Width(18).Render(h.IP)

					port := h.Port
					if port == 0 {
						port = 22
					}
					portRow := rowStyle.Width(8).Render(fmt.Sprintf("%d", port))

					user := h.User
					if user == "" {
						user = "root"
					}
					userRow := rowStyle.Width(12).Render(user)

					var authMethod string
					if h.SSHPrivateKey != "" && h.SSHPassword != "" {
						authMethod = fmt.Sprintf("Key (%s) + Password", h.SSHPrivateKey)
					} else if h.SSHPrivateKey != "" {
						authMethod = fmt.Sprintf("Key (%s)", h.SSHPrivateKey)
					} else if h.SSHPassword != "" {
						authMethod = "Password"
					} else {
						authMethod = "Default (Key/Agent)"
					}
					authRow := rowStyle.Width(38).Render(authMethod)

					py := h.PythonInterpreter
					if py == "" {
						py = "default"
					}
					pyRow := rowStyle.Width(22).Render(py)

					cmd.Println(lipgloss.JoinHorizontal(lipgloss.Top, gRow, ipRow, portRow, userRow, authRow, pyRow))
				}
			}

			cmd.Println()
			cmd.Printf("Total: %d groups, %d servers\n", totalGroups, totalHosts)
			return nil
		},
	}
	cmd.Flags().StringVarP(&group, "group", "g", "all", "filter target host group to list")
	return cmd
}
