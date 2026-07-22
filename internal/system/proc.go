package system

import (
	"bytes"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// FindPID returns the PID of the oldest running process matching name.
// On Linux/macOS it uses pgrep; on Windows it falls back to tasklist.
func FindPID(name string) (int, error) {
	if runtime.GOOS == "windows" {
		return findPIDWindows(name)
	}
	cmd := exec.Command("pgrep", "-o", name)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(bytes.TrimSpace(out))))
	if err != nil {
		return 0, err
	}
	return pid, nil
}

// findPIDWindows uses tasklist to find the first PID of a process by name.
func findPIDWindows(name string) (int, error) {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq "+name+".exe", "/FO", "CSV", "/NH")
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	// CSV output example: "nginx.exe","1234","Console","1","5,456 K"
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) >= 2 {
			pidStr := strings.Trim(fields[1], "\"")
			if pid, err := strconv.Atoi(pidStr); err == nil && pid > 0 {
				return pid, nil
			}
		}
	}
	return 0, exec.ErrNotFound
}
