package ansiblecmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"OpsVault/internal/driver/ansible"

	"github.com/spf13/cobra"
)

func (c *commandSet) newPushCommand() *cobra.Command {
	var group string
	var binPath string
	var cfgPath string

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push OpsVault executable binary and default.yaml config to remote hosts",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Running pre-command: make all...")
			makeCmd := exec.CommandContext(cmd.Context(), "make", "all")
			makeCmd.Stdout = os.Stdout
			makeCmd.Stderr = os.Stderr
			if err := makeCmd.Run(); err != nil {
				return fmt.Errorf("pre-command 'make all' failed: %w", err)
			}

			// Convert binary path to absolute path so Ansible resolves it correctly from root directory
			absBinPath, err := filepath.Abs(binPath)
			if err != nil {
				return fmt.Errorf("failed to get absolute path for binary %s: %w", binPath, err)
			}
			binPath = absBinPath

			if _, err := os.Stat(binPath); err != nil {
				if strings.HasSuffix(binPath, "/bin/opsvault-linux-amd64") || binPath == "./bin/opsvault-linux-amd64" {
					fallbackBin := filepath.Join(filepath.Dir(binPath), "opsvault")
					if _, errFallback := os.Stat(fallbackBin); errFallback == nil {
						binPath = fallbackBin
					} else {
						return fmt.Errorf("binary file not found at %s: please ensure 'make all' compiled the Linux executable", binPath)
					}
				} else {
					return fmt.Errorf("binary file not found at %s: %w", binPath, err)
				}
			}

			// Convert config path to absolute path so Ansible resolves it correctly from root directory
			absCfgPath, err := filepath.Abs(cfgPath)
			if err != nil {
				return fmt.Errorf("failed to get absolute path for config %s: %w", cfgPath, err)
			}
			cfgPath = absCfgPath

			if _, err := os.Stat(cfgPath); err != nil {
				return fmt.Errorf("config file not found at %s: %w", cfgPath, err)
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
			tempDir := v.GetString("ansible.temp_dir")
			if tempDir == "" {
				tempDir = "/data/opsvault/ansible/tmp"
			}

			vars := ansible.PlaybookVars{
				TargetGroup: group,
				DataRoot:    dataRoot,
				BinaryPath:  binPath,
				ConfigPath:  cfgPath,
			}

			fmt.Printf("Generating push playbook for group: %s...\n", group)
			playbookFile, err := ansible.GeneratePlaybookFile(tempDir, "push", vars)
			if err != nil {
				return fmt.Errorf("failed to generate push playbook: %w", err)
			}
			defer func() {
				_ = os.Remove(playbookFile)
			}()

			fmt.Printf("Pushing OpsVault binary (%s) and config (%s) to group: %s...\n", binPath, cfgPath, group)
			err = exec.RunPlaybook(cmd.Context(), playbookFile, group, nil, os.Stdout, os.Stderr)
			if err != nil {
				return fmt.Errorf("push playbook execution failed: %w", err)
			}

			fmt.Printf("Successfully pushed binary and configuration to target group: %s.\n", group)
			return nil
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "all", "target host group for push")
	cmd.Flags().StringVar(&binPath, "bin", "./bin/opsvault-linux-amd64", "local path to OpsVault executable binary")
	cmd.Flags().StringVar(&cfgPath, "config-path", "./configs/default.yaml", "local path to configuration file to push")

	return cmd
}
