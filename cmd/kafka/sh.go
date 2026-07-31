package kafka

import (
	"OpsVault/pkg/dockercli"
	"OpsVault/pkg/executil"

	"github.com/spf13/cobra"
)

func (c *commandSet) newShCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sh",
		Short: "Open an interactive shell inside the Kafka container",
		Long:  "Connects to the Kafka container with a bash shell.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			containerName := dockercli.ResolveContainerName(c.config, "kafka")
			return executil.DockerExec(containerName, []string{"bash"})
		},
	}
}
