package mysql

import (
	"context"
	"io"

	"github.com/spf13/cobra"
)

func (c *commandSet) newLogCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "log",
		Short: "Show MySQL logs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			drv, err := c.driver("")
			if err != nil {
				return err
			}
			if drv.Client == nil {
				cmd.Println("docker client unavailable")
				return nil
			}
			reader, err := drv.Client.ContainerLogs(context.Background(), drv.ContainerName, containerLogsOptions())
			if err != nil {
				return err
			}
			defer reader.Close()
			_, err = io.Copy(cmd.OutOrStdout(), reader)
			return err
		},
	}
}
