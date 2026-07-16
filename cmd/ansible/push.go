package ansiblecmd

import (
	"fmt"
	"os"

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
			if _, err := os.Stat(binPath); err != nil {
				if binPath == "./bin/opsvault-linux-amd64" {
					if _, errFallback := os.Stat("./bin/opsvault"); errFallback == nil {
						binPath = "./bin/opsvault"
					} else {
						return fmt.Errorf("binary file not found at %s (or ./bin/opsvault): please compile the Linux executable first using: GOOS=linux GOARCH=amd64 go build -o bin/opsvault-linux-amd64 main.go", binPath)
					}
				} else {
					return fmt.Errorf("binary file not found at %s: %w", binPath, err)
				}
			}

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
