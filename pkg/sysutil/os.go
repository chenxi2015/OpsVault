package sysutil

import (
	"os"
	"runtime"
)

// IsLinux checks if the current OS is Linux.
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// IsRoot checks if the current process runs with root privileges.
func IsRoot() bool {
	return os.Getuid() == 0
}
