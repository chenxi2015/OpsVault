package mysql

import (
	"OpsVault/cmd/common"
	"OpsVault/internal/driver"

	"github.com/spf13/cobra"
)

func (c *commandSet) newStartCommand() *cobra.Command {
	return lifecycleCommand(c, "start", "Start MySQL", func(d dockerDriverShim) error { return d.Start() })
}

type dockerDriverShim interface {
	Start() error
	Stop() error
	Restart() error
	Uninstall(bool) error
	Upgrade(string) error
	Status() (*driver.ServiceStatus, error)
}

func lifecycleCommand(c *commandSet, use, short string, fn func(dockerDriverShim) error) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := common.RequireMode(driver.Mode(c.config.GetString("mode")), driver.ModeDocker); err != nil {
				return err
			}
			drv, err := c.driver("")
			if err != nil {
				return err
			}
			var shim dockerDriverShim = drv
			return fn(shim)
		},
	}
}
