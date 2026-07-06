package system

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"OpsVault/pkg/sysutil"
)

// ApplyULimit applies limit settings and tunes system parameters.
func ApplyULimit() error {
	if !sysutil.IsLinux() {
		// Friendly skip for non-Linux OS (macOS/Windows developer machine)
		return nil
	}

	if !sysutil.IsRoot() {
		return fmt.Errorf("root privileges are required to configure system limits and kernel parameters")
	}

	// 1. Configure /etc/security/limits.conf
	limitsPath := "/etc/security/limits.conf"
	limitsConfig := []string{
		"* soft nofile 1000000",
		"* hard nofile 1000000",
	}
	if err := appendConfigIfNotExist(limitsPath, limitsConfig); err != nil {
		return fmt.Errorf("configure limits.conf: %w", err)
	}

	// 2. Configure /etc/sysctl.conf
	sysctlPath := "/etc/sysctl.conf"
	sysctlConfig := []string{
		"fs.file-max = 1000000",
		"net.ipv4.tcp_max_syn_backlog = 65535",
	}
	if err := appendConfigIfNotExist(sysctlPath, sysctlConfig); err != nil {
		return fmt.Errorf("configure sysctl.conf: %w", err)
	}

	// 3. Apply sysctl settings
	if out, err := exec.Command("sysctl", "-p").CombinedOutput(); err != nil {
		return fmt.Errorf("apply sysctl settings: %w: %s", err, string(out))
	}

	// 4. Temporarily raise current process limits using prlimit (Linux specific tool)
	pid := os.Getpid()
	prlimitCmd := exec.Command("prlimit", fmt.Sprintf("--pid=%d", pid), "--nofile=1000000:1000000")
	_ = prlimitCmd.Run() // non-critical error, do not fail

	return nil
}

// appendConfigIfNotExist appends lines to a file if they don't already exist.
func appendConfigIfNotExist(path string, lines []string) error {
	data, err := os.ReadFile(path)
	var content string
	if err == nil {
		content = string(data)
	} else if !os.IsNotExist(err) {
		return err
	}

	var toAppend []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Search exact line
		found := false
		for _, fileLine := range strings.Split(content, "\n") {
			if strings.TrimSpace(fileLine) == trimmed {
				found = true
				break
			}
		}
		if !found {
			toAppend = append(toAppend, line)
		}
	}

	if len(toAppend) == 0 {
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if !strings.HasSuffix(content, "\n") && len(content) > 0 {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}

	for _, line := range toAppend {
		if _, err := f.WriteString(line + "\n"); err != nil {
			return err
		}
	}

	return nil
}
