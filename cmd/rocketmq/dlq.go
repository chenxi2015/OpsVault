package rocketmq

import "github.com/spf13/cobra"

func (c *commandSet) newDLQCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "dlq", Short: "Dead-letter queue tools"}
	cmd.AddCommand(&cobra.Command{
		Use:   "stat",
		Short: "Show DLQ stats",
		RunE: func(cmd *cobra.Command, _ []string) error {
			drv, err := c.driver()
			if err != nil {
				return err
			}
			for key, value := range drv.DLQStat() {
				cmd.Printf("%s: %s\n", key, value)
			}
			return nil
		},
	})
	return cmd
}
