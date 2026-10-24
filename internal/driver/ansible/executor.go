package ansible

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"OpsVault/pkg/logger"
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
	return e.RunAnsibleWithOpts(ctx, group, module, args, AdhocOpts{}, stdout, stderr)
}

// AdhocOpts carries optional modifiers for an ad-hoc ansible run.
type AdhocOpts struct {
	// Limit is passed to ansible -l to restrict the run to specific hosts.
	Limit string
	// TimeoutSec is passed as a shell-level `timeout <s>` wrapper when > 0.
	TimeoutSec int
}

// RunAnsibleWithOpts runs an ad-hoc ansible command with optional limit/timeout.
func (e *Executor) RunAnsibleWithOpts(ctx context.Context, group string, module string, args string, opts AdhocOpts, stdout, stderr io.Writer) error {
	if err := EnsureAnsible(e.Cfg); err != nil {
		return err
	}
	logger.Infof("[ansible] Executing ad-hoc command (group=%s, module=%s, limit=%q, timeout=%ds)...",
		group, module, opts.Limit, opts.TimeoutSec)
	cmdArgs := []string{
		group,
		"-i", e.InventoryPath,
		"-m", module,
	}
	if opts.Limit != "" {
		cmdArgs = append(cmdArgs, "-l", opts.Limit)
	}
	if args != "" {
		if opts.TimeoutSec > 0 && module == "shell" {
			// Wrap with coreutils `timeout` so a hung remote command cannot
			// hold the ansible connection open forever. Signal is SIGTERM
			// first; the remote `timeout` will escalate per its defaults.
			args = fmt.Sprintf("timeout %d bash -c %s", opts.TimeoutSec, shellQuote(args))
		}
		cmdArgs = append(cmdArgs, "-a", args)
	}

	cmd := exec.CommandContext(ctx, e.Cfg.BinPath, cmdArgs...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	// Add env to bypass host key checking just in case
	cmd.Env = append(os.Environ(), "ANSIBLE_HOST_KEY_CHECKING=False")

	if err := cmd.Run(); err != nil {
		logger.Errorf("[ansible] Ad-hoc command failed: %v", err)
		return fmt.Errorf("ansible run failed: %w", err)
	}
	logger.Infof("[ansible] Ad-hoc command completed successfully")
	return nil
}

// shellQuote wraps s in single quotes with proper escaping for the remote shell.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// RunPlaybook runs an ansible-playbook command on the specified target group.
func (e *Executor) RunPlaybook(ctx context.Context, playbookPath string, group string, extraVars map[string]string, stdout, stderr io.Writer) error {
	if err := EnsureAnsible(e.Cfg); err != nil {
		return err
	}
	logger.Infof("[ansible] Running playbook %s (group=%s)...", playbookPath, group)
	cmdArgs := []string{
		"-i", e.InventoryPath,
		playbookPath,
	}
	if group != "" && group != "all" {
		cmdArgs = append(cmdArgs, "-l", group)
	}

	for k, v := range extraVars {
		cmdArgs = append(cmdArgs, "--extra-vars", fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.CommandContext(ctx, e.Cfg.PlaybookBinPath, cmdArgs...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(), "ANSIBLE_HOST_KEY_CHECKING=False")

	if err := cmd.Run(); err != nil {
		logger.Errorf("[ansible] Playbook execution failed: %v", err)
		return fmt.Errorf("ansible-playbook run failed: %w", err)
	}
	logger.Infof("[ansible] Playbook %s executed successfully", playbookPath)
	return nil
}
