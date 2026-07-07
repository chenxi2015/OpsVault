package elk

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (c *commandSet) newLogCommand() *cobra.Command {
	var component string
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show ELK Stack component logs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			drv, err := c.driver()
			if err != nil {
				return err
			}
			out, err := drv.TailComponentLogs(component, 100)
			if err != nil {
				return err
			}
			fmt.Print(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&component, "component", "elasticsearch", "ELK component to read logs from: elasticsearch|kibana|logstash")
	return cmd
}
