package executil

import (
	"fmt"
	"os"
	"os/exec"
)

// DockerExec runs an interactive docker exec command in the named container.
// It attaches the current process's stdin/stdout/stderr so the user gets a full TTY.
func DockerExec(containerName string, cmdArgs []string) error {
	// Check docker binary exists
	dockerBin, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("docker not found in PATH: %w", err)
	}

	args := append([]string{"exec", "-it", containerName}, cmdArgs...)
	cmd := exec.Command(dockerBin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
