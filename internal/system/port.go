package system

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// CheckPortAvailable checks if a TCP port is available to listen on.
func CheckPortAvailable(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	return ln.Close()
}

// GetPortOccupant tries to find the PID and process name occupying the given TCP port.
// It returns pid, processName, and an error if it could not retrieve the information.
func GetPortOccupant(port int) (int, string, error) {
	// 1. Try lsof command first (works on both macOS and Linux if installed)
	if _, err := exec.LookPath("lsof"); err == nil {
		cmd := exec.Command("lsof", "-iTCP:"+strconv.Itoa(port), "-sTCP:LISTEN", "-P", "-n", "-F", "pc")
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			output := out.String()
			lines := strings.Split(output, "\n")
			var pid int
			var procName string
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "p") {
					if p, err := strconv.Atoi(line[1:]); err == nil {
						pid = p
					}
				} else if strings.HasPrefix(line, "c") {
					procName = line[1:]
				}
			}
			if pid > 0 && procName != "" {
				return pid, procName, nil
			}
		}
	}

	// 2. Try ss command (standard on Linux/CentOS)
	if _, err := exec.LookPath("ss"); err == nil {
		cmd := exec.Command("ss", "-tlnp", "sport = :"+strconv.Itoa(port))
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			output := out.String()
			// Example output line:
			// LISTEN     0      128          *:3306                     *:*                   users:(("mysqld",pid=1234,fd=6))
			// Or: users:(("mysqld",1234,6))
			re := regexp.MustCompile(`(?i)users:\(\("([^"]+)",(?:pid=)?(\d+)`)
			matches := re.FindStringSubmatch(output)
			if len(matches) >= 3 {
				procName := matches[1]
				if pid, err := strconv.Atoi(matches[2]); err == nil {
					return pid, procName, nil
				}
			}
		}
	}

	// 3. Try netstat command as fallback
	if _, err := exec.LookPath("netstat"); err == nil {
		cmd := exec.Command("netstat", "-tlpn")
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			// Search for the port in the output
			lines := strings.Split(out.String(), "\n")
			portStr := fmt.Sprintf(":%d", port)
			for _, line := range lines {
				if strings.Contains(line, "LISTEN") && strings.Contains(line, portStr) {
					// Output looks like:
					// tcp        0      0 0.0.0.0:3306            0.0.0.0:*               LISTEN      1234/mysqld
					fields := strings.Fields(line)
					if len(fields) >= 7 {
						lastField := fields[len(fields)-1]
						if parts := strings.Split(lastField, "/"); len(parts) == 2 {
							if pid, err := strconv.Atoi(parts[0]); err == nil {
								return pid, parts[1], nil
							}
						}
					}
				}
			}
		}
	}

	return 0, "", fmt.Errorf("port occupant not found")
}

