//go:build !windows

package sysutil

// isWindowsAdmin is not applicable on non-Windows platforms.
func isWindowsAdmin() bool {
	return false
}
