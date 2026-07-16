package ansiblecmd

import (
	"errors"
	"fmt"
	"os"

	"OpsVault/internal/driver/ansible"

	"github.com/spf13/cobra"
)

func (c *commandSet) newUninstallCommand() *cobra.Command {
	var service string
	var group string
	var purge bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall middleware service or Docker from remote hosts via Ansible",
		RunE: func(cmd *cobra.Command, args []string) error {
			if service == "" {
				return errors.New("service name must be specified, use --service")
			}

			switch service {
			case "docker", "mysql", "redis", "rabbitmq", "nginx":
				// valid
			default:
				return fmt.Errorf("unsupported service: %s. Supported: docker, mysql, redis, rabbitmq, nginx", service)
			}

			exec, cleanup, err := c.getExecutor()
			if err != nil {
				return err
			}
			defer cleanup()

			v := c.config
			dataRoot := v.GetString("docker.data_root")
			if dataRoot == "" {
				dataRoot = "/data/opsvault"
			}
			namePrefix := v.GetString("docker.name_prefix")
			if namePrefix == "" {
				namePrefix = "opsvault"
			}
			tempDir := v.GetString("ansible.temp_dir")
			if tempDir == "" {
				tempDir = "/data/opsvault/ansible/tmp"
			}

			vars := ansible.PlaybookVars{
				TargetGroup: group,
				DataRoot:    dataRoot,
				NamePrefix:  namePrefix,
				Purge:       purge,
				ServiceName: service,
				// Populate Nginx paths from config for accurate uninstallation/purging
				NginxInstallPath: v.GetString("nginx.install_path"),
				NginxWWWRoot:     v.GetString("nginx.www_root"),
				NginxSSLRoot:     v.GetString("nginx.ssl_root"),
				NginxWWWLogsRoot: v.GetString("nginx.wwwlogs_root"),
			}

			fmt.Printf("Generating uninstall playbook for service: %s (purge=%v)...\n", service, purge)
			playbookFile, err := ansible.GenerateUninstallPlaybookFile(tempDir, service, vars)
			if err != nil {
				return fmt.Errorf("failed to generate uninstall playbook: %w", err)
			}
			defer func() {
				_ = os.Remove(playbookFile)
			}()

			fmt.Printf("Executing uninstall playbook on group: %s...\n", group)
			err = exec.RunPlaybook(cmd.Context(), playbookFile, group, nil, os.Stdout, os.Stderr)
			if err != nil {
				return fmt.Errorf("uninstall playbook execution failed: %w", err)
			}

			fmt.Printf("Uninstallation of %s completed successfully on group: %s.\n", service, group)
			return nil
		},
	}

	cmd.Flags().StringVarP(&service, "service", "s", "", "middleware service to uninstall (docker, mysql, redis, rabbitmq, nginx)")
	cmd.Flags().StringVarP(&group, "group", "g", "all", "target host group for uninstallation")
	cmd.Flags().BoolVar(&purge, "purge", false, "completely purge data volumes, config files, and website directories")
	_ = cmd.MarkFlagRequired("service")

	return cmd
}
