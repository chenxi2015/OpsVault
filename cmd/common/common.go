package common

import (
	"fmt"
	"sort"
	"strings"

	"OpsVault/internal/driver"

	"github.com/spf13/cobra"
)

func PrintStatus(cmd *cobra.Command, status *driver.ServiceStatus) {
	if status == nil {
		cmd.Println("no status available")
		return
	}
	cmd.Printf("name: %s\n", status.Name)
	cmd.Printf("mode: %s\n", status.Mode)
	cmd.Printf("status: %s\n", status.Status)
	cmd.Printf("running: %t\n", status.Running)
	if status.Version != "" {
		cmd.Printf("version: %s\n", status.Version)
	}
	if len(status.Ports) > 0 {
		cmd.Printf("ports: %s\n", strings.Join(status.Ports, ", "))
	}
	if status.DataPath != "" {
		cmd.Printf("data_path: %s\n", status.DataPath)
	}
	if status.Network != "" {
		cmd.Printf("network: %s\n", status.Network)
	}
	if status.PID > 0 {
		cmd.Printf("pid: %d\n", status.PID)
	}
	if len(status.Details) > 0 {
		keys := make([]string, 0, len(status.Details))
		for key := range status.Details {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			cmd.Printf("%s: %s\n", key, status.Details[key])
		}
	}
}

func RequireMode(actual driver.Mode, allowed ...driver.Mode) error {
	for _, item := range allowed {
		if actual == item {
			return nil
		}
	}
	values := make([]string, 0, len(allowed))
	for _, item := range allowed {
		values = append(values, string(item))
	}
	return fmt.Errorf("unsupported mode %q, allowed: %s", actual, strings.Join(values, ", "))
}
