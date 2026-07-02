package rocketmq

import (
	"context"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/spf13/cobra"
)

func (c *commandSet) newLogCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "log",
		Short: "Show RocketMQ logs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			drv, err := c.driver()
			if err != nil {
				return err
			}
			if drv.Client == nil {
				cmd.Println("docker client unavailable")
				return nil
			}
			reader, err := drv.Client.ContainerLogs(context.Background(), drv.ContainerName, container.LogsOptions{ShowStdout: true, ShowStderr: true, Tail: "100"})
			if err != nil {
				return err
			}
			defer reader.Close()
			_, err = io.Copy(cmd.OutOrStdout(), reader)
			return err
		},
	}
}
