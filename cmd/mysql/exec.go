package mysql

import (
	"OpsVault/pkg/executil"

	"github.com/spf13/cobra"
)

func (c *commandSet) newExecCommand() *cobra.Command {
	var user string
	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Open an interactive MySQL shell inside the container",
		Long:  "Connects to the opsvault-mysql container and opens a mysql interactive shell.\nEquivalent to: docker exec -it opsvault-mysql mysql -u <user> -p",
		RunE: func(cmd *cobra.Command, _ []string) error {
			password := c.config.GetString("mysql.root_password")
			args := []string{"mysql", "-u", user}
			if password != "" {
				args = append(args, "-p"+password)
			} else {
				// Prompt for password interactively
				args = append(args, "-p")
			}
			return executil.DockerExec("opsvault-mysql", args)
		},
	}
	cmd.Flags().StringVarP(&user, "user", "u", "root", "MySQL user to connect as")
	return cmd
}
