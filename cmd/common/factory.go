package common

import (
	"fmt"
	"io"
	"os"

	"OpsVault/internal/driver"

	"github.com/spf13/cobra"
)

// DriverProvider is a function type that returns a ServiceDriver instance.
type DriverProvider func() (driver.ServiceDriver, error)

// LogDriver is an interface for drivers that support tailing logs.
type LogDriver interface {
	TailLogs(lines int, follow bool, out io.Writer) error
}

// LogDriverProvider is a function type that returns a LogDriver instance.
type LogDriverProvider func() (LogDriver, error)

// NewStartCmd creates a standard start cobra command for a service.
func NewStartCmd(serviceName string, getMode func() string, getDriver DriverProvider) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: fmt.Sprintf("Start %s service", serviceName),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := RequireMode(driver.Mode(getMode()), driver.ModeDocker, driver.ModeBinary); err != nil {
				return err
			}
			drv, err := getDriver()
			if err != nil {
				return err
			}
			return drv.Start()
		},
	}
}

// NewStopCmd creates a standard stop cobra command for a service.
func NewStopCmd(serviceName string, getMode func() string, getDriver DriverProvider) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: fmt.Sprintf("Stop %s service", serviceName),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := RequireMode(driver.Mode(getMode()), driver.ModeDocker, driver.ModeBinary); err != nil {
				return err
			}
			drv, err := getDriver()
			if err != nil {
				return err
			}
			return drv.Stop()
		},
	}
}

// NewRestartCmd creates a standard restart cobra command for a service.
func NewRestartCmd(serviceName string, getMode func() string, getDriver DriverProvider) *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: fmt.Sprintf("Restart %s service", serviceName),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := RequireMode(driver.Mode(getMode()), driver.ModeDocker, driver.ModeBinary); err != nil {
				return err
			}
			drv, err := getDriver()
			if err != nil {
				return err
			}
			return drv.Restart()
		},
	}
}

// NewStatusCmd creates a standard status cobra command for a service.
func NewStatusCmd(serviceName string, getMode func() string, getDriver DriverProvider) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: fmt.Sprintf("Check %s service status", serviceName),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := RequireMode(driver.Mode(getMode()), driver.ModeDocker, driver.ModeBinary); err != nil {
				return err
			}
			drv, err := getDriver()
			if err != nil {
				return err
			}
			st, err := drv.Status()
			if err != nil {
				return err
			}
			PrintStatus(cmd, st)
			return nil
		},
	}
}

// NewLogCmd creates a standard log cobra command for a service.
func NewLogCmd(serviceName string, getMode func() string, getLogDriver LogDriverProvider) *cobra.Command {
	var follow bool
	var lines int
	cmd := &cobra.Command{
		Use:   "log",
		Short: fmt.Sprintf("View %s service logs", serviceName),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := RequireMode(driver.Mode(getMode()), driver.ModeDocker, driver.ModeBinary); err != nil {
				return err
			}
			drv, err := getLogDriver()
			if err != nil {
				return err
			}
			return drv.TailLogs(lines, follow, os.Stdout)
		},
	}
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	cmd.Flags().IntVarP(&lines, "lines", "n", 100, "Output specified number of lines")
	return cmd
}

// NewUninstallCmd creates a standard uninstall cobra command for a service.
func NewUninstallCmd(serviceName string, getMode func() string, getDriver DriverProvider) *cobra.Command {
	var purge bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: fmt.Sprintf("Uninstall %s service", serviceName),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := RequireMode(driver.Mode(getMode()), driver.ModeDocker, driver.ModeBinary); err != nil {
				return err
			}
			drv, err := getDriver()
			if err != nil {
				return err
			}
			return drv.Uninstall(purge)
		},
	}
	cmd.Flags().BoolVar(&purge, "purge", false, "Purge data directory")
	return cmd
}
