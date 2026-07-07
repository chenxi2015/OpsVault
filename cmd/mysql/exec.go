package mysql

import (
	"OpsVault/pkg/dockercli"
	"OpsVault/pkg/executil"

	"github.com/spf13/cobra"
)

func (c *commandSet) newExecCommand() *cobra.Command {
	var user string
	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Open an interactive MySQL shell inside the container",
		Long:  "Connects to the MySQL container and opens a mysql interactive shell.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			password := c.config.GetString("mysql.root_password")
			args := []string{"mysql", "-u", user}
			if password != "" {
				args = append(args, "-p"+password)
			} else {
				// Prompt for password interactively
				args = append(args, "-p")
			}
			containerName := dockercli.ResolveContainerName(c.config, "mysql")
			return executil.DockerExec(containerName, args)
		},
	}
	cmd.Flags().StringVarP(&user, "user", "u", "root", "MySQL user to connect as")
	return cmd
}
