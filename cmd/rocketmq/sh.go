package rocketmq

import (
	"OpsVault/pkg/executil"

	"github.com/spf13/cobra"
)

const rocketMQToolsPath = "/home/rocketmq/rocketmq/bin/tools.sh"

func (c *commandSet) newShCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sh",
		Short: "Open an interactive shell inside the RocketMQ container",
		Long:  "Connects to the opsvault-rocketmq container with a bash shell.\nUse 'mqadmin' commands inside for advanced operations.\nEquivalent to: docker exec -it opsvault-rocketmq bash",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executil.DockerExec("opsvault-rocketmq", []string{"bash"})
		},
	}
	return cmd
}
