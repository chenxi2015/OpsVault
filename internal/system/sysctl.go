package system

import "os/exec"

func ApplyULimit() error {
	cmd := exec.Command("bash", "-lc", "ulimit -n 65535")
	return cmd.Run()
}
