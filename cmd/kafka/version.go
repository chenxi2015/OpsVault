package kafka

import "github.com/spf13/cobra"

func (c *commandSet) newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show Kafka image/version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			drv, err := c.driver()
			if err != nil {
				return err
			}
			cmd.Println(drv.Version())
			return nil
		},
	}
}
