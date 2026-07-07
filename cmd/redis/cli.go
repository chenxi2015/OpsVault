package redis

import (
	"OpsVault/pkg/dockercli"
	"OpsVault/pkg/executil"

	"github.com/spf13/cobra"
)

func (c *commandSet) newCliCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cli",
		Short: "Open an interactive Redis CLI inside the container",
		Long:  "Connects to the Redis container and opens redis-cli.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			args := []string{"redis-cli"}
			password := c.config.GetString("redis.password")
			if password != "" {
				args = append(args, "-a", password)
			}
			containerName := dockercli.ResolveContainerName(c.config, "redis")
			return executil.DockerExec(containerName, args)
		},
	}
	return cmd
}
