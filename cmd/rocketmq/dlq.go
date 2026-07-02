package rocketmq

import (
	"sort"

	"github.com/spf13/cobra"
)

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
			stats, err := drv.DLQStat()
			if err != nil {
				return err
			}
			keys := make([]string, 0, len(stats))
			for key := range stats {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				value := stats[key]
				cmd.Printf("%s: %s\n", key, value)
			}
			return nil
		},
	})
	return cmd
}
