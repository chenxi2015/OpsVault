package ansiblecmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func (c *commandSet) newExecCommand() *cobra.Command {
	var group string
	var commandStr string
	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Execute an ad-hoc shell command on target hosts",
		RunE: func(cmd *cobra.Command, args []string) error {
			if commandStr == "" {
				return errors.New("command string must not be empty, use --cmd")
			}
			exec, cleanup, err := c.getExecutor()
			if err != nil {
				return err
			}
			defer cleanup()

			fmt.Printf("Executing command on group %s: %s\n", group, commandStr)
			err = exec.RunAnsible(cmd.Context(), group, "shell", commandStr, os.Stdout, os.Stderr)
			if err != nil {
				return fmt.Errorf("execution failed: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&group, "group", "g", "all", "target host group")
	cmd.Flags().StringVarP(&commandStr, "cmd", "c", "", "shell command to execute")
	_ = cmd.MarkFlagRequired("cmd")
	return cmd
}
