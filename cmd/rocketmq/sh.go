package rocketmq

import (
	"OpsVault/pkg/dockercli"
	"OpsVault/pkg/executil"

	"github.com/spf13/cobra"
)

const rocketMQToolsPath = "/home/rocketmq/rocketmq/bin/tools.sh"

func (c *commandSet) newShCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sh",
		Short: "Open an interactive shell inside the RocketMQ container",
		Long:  "Connects to the RocketMQ container with a bash shell.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			containerName := dockercli.ResolveContainerName(c.config, "rocketmq")
			return executil.DockerExec(containerName, []string{"bash"})
		},
	}
	return cmd
}
