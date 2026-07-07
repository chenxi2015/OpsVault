package redis

import (
	"OpsVault/pkg/executil"

	"github.com/spf13/cobra"
)

func (c *commandSet) newCliCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cli",
		Short: "Open an interactive Redis CLI inside the container",
		Long:  "Connects to the opsvault-redis container and opens redis-cli.\nEquivalent to: docker exec -it opsvault-redis redis-cli [-a <password>]",
		RunE: func(cmd *cobra.Command, _ []string) error {
			args := []string{"redis-cli"}
			password := c.config.GetString("redis.password")
			if password != "" {
				args = append(args, "-a", password)
			}
			return executil.DockerExec("opsvault-redis", args)
		},
	}
	return cmd
}
