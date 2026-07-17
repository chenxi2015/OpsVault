package nacos

import (
	"fmt"
	"strings"

	"OpsVault/cmd/common"
	"OpsVault/internal/driver"

	"github.com/spf13/cobra"
)

func (c *commandSet) newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show Nacos status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver("")
			if err != nil {
				return err
			}
			status, err := drv.Status()
			if err != nil {
				return err
			}
			fmt.Printf("Service:      %s\n", status.Name)
			fmt.Printf("Mode:         %s\n", status.Mode)
			fmt.Printf("Running:      %v\n", status.Running)
			fmt.Printf("Status:       %s\n", status.Status)
			fmt.Printf("Data Path:    %s\n", status.DataPath)
			if len(status.Ports) > 0 {
				fmt.Printf("Ports:        %s\n", strings.Join(status.Ports, ", "))
			}
			if status.Network != "" {
				fmt.Printf("Network:      %s\n", status.Network)
			}
			for k, v := range status.Details {
				fmt.Printf("%-13s %s\n", k+":", v)
			}
			return nil
		},
	}
}
