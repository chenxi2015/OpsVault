package ansiblecmd

import (
	"errors"
	"fmt"
	"os"

	"OpsVault/internal/driver/ansible"

	"github.com/spf13/cobra"
)

// newReloadCommand creates a subcommand to reload middleware services on remote hosts.
func (c *commandSet) newReloadCommand() *cobra.Command {
	var service string
	var group string

	cmd := &cobra.Command{
		Use:   "reload",
		Short: "Reload configuration of middleware service on remote hosts via Ansible",
		RunE: func(cmd *cobra.Command, args []string) error {
			if service == "" {
				return errors.New("service name must be specified, use --service")
			}

			switch service {
			case "nginx":
				// valid
			default:
				return fmt.Errorf("unsupported service for reload: %s. Supported: nginx", service)
			}

			exec, cleanup, err := c.getExecutor()
			if err != nil {
				return err
			}
			defer cleanup()

			v := c.config
			tempDir := v.GetString("ansible.temp_dir")
			if tempDir == "" {
				tempDir = "/data/opsvault/ansible/tmp"
			}

			vars := ansible.PlaybookVars{
				TargetGroup: group,
				ServiceName: service,
			}

			fmt.Printf("Generating reload playbook for service: %s...\n", service)
			playbookFile, err := ansible.GenerateReloadPlaybookFile(tempDir, service, vars)
			if err != nil {
				return fmt.Errorf("failed to generate reload playbook: %w", err)
			}
			defer func() {
				_ = os.Remove(playbookFile)
			}()

			fmt.Printf("Executing reload playbook on group: %s...\n", group)
			err = exec.RunPlaybook(cmd.Context(), playbookFile, group, nil, os.Stdout, os.Stderr)
			if err != nil {
				return fmt.Errorf("reload playbook execution failed: %w", err)
			}

			fmt.Printf("Reload of %s completed successfully on group: %s.\n", service, group)
			return nil
		},
	}

	cmd.Flags().StringVarP(&service, "service", "s", "", "middleware service to reload (nginx)")
	cmd.Flags().StringVarP(&group, "group", "g", "all", "target host group for reload")
	_ = cmd.MarkFlagRequired("service")

	return cmd
}
