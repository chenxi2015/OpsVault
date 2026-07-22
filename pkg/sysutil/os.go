package sysutil

import (
	"os"
	"runtime"
)

// IsLinux checks if the current OS is Linux.
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// IsWindows checks if the current OS is Windows.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsRoot checks if the current process runs with elevated privileges.
// On Linux/macOS it checks UID == 0; on Windows it checks for Admin token elevation.
func IsRoot() bool {
	if runtime.GOOS == "windows" {
		return isWindowsAdmin()
	}
	return os.Getuid() == 0
}
