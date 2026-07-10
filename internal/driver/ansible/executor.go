package ansible

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Executor is responsible for running ansible CLI commands.
type Executor struct {
	Cfg           *Config
	InventoryPath string
}

// NewExecutor initializes an Executor with a configuration and inventory file.
func NewExecutor(cfg *Config, inventoryPath string) *Executor {
	return &Executor{
		Cfg:           cfg,
		InventoryPath: inventoryPath,
	}
}

// RunAnsible runs the basic ansible ad-hoc command.
func (e *Executor) RunAnsible(ctx context.Context, group string, module string, args string, stdout, stderr io.Writer) error {
	cmdArgs := []string{
		group,
		"-i", e.InventoryPath,
		"-m", module,
	}
	if args != "" {
		cmdArgs = append(cmdArgs, "-a", args)
	}

	cmd := exec.CommandContext(ctx, e.Cfg.BinPath, cmdArgs...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	// Add env to bypass host key checking just in case
	cmd.Env = append(os.Environ(), "ANSIBLE_HOST_KEY_CHECKING=False")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ansible run failed: %w", err)
	}
	return nil
}

// RunPlaybook runs an ansible-playbook command.
func (e *Executor) RunPlaybook(ctx context.Context, playbookPath string, extraVars map[string]string, stdout, stderr io.Writer) error {
	cmdArgs := []string{
		"-i", e.InventoryPath,
		playbookPath,
	}

	for k, v := range extraVars {
		cmdArgs = append(cmdArgs, "--extra-vars", fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.CommandContext(ctx, e.Cfg.PlaybookBinPath, cmdArgs...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(), "ANSIBLE_HOST_KEY_CHECKING=False")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ansible-playbook run failed: %w", err)
	}
	return nil
}
