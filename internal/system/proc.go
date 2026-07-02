package system

import (
	"bytes"
	"os/exec"
	"strconv"
	"strings"
)

func FindPID(name string) (int, error) {
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
