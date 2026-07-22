//go:build windows

package sysutil

import (
	"os/exec"
)

// isWindowsAdmin checks if the current process has administrator privileges
// by attempting a privileged operation (reading the SAM key via reg query).
// This approach has zero external dependencies.
func isWindowsAdmin() bool {
	// "reg query HKU\S-1-5-19" succeeds only when running as admin/elevated.
	err := exec.Command("reg", "query", `HKU\S-1-5-19`).Run()
	return err == nil
}
