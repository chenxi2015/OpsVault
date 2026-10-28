package ansiblecmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"OpsVault/internal/driver/ansible"
	"OpsVault/pkg/logger"
	"OpsVault/pkg/safety"

	"github.com/spf13/cobra"
)

// execTarget encapsulates the resolved target for ad-hoc execution.
type execTarget struct {
	displayName string
	content     string
	module      string
	moduleArgs  string
	auditDetail string
	isScript    bool
}

// resolveExecTarget validates and parses command or script file inputs.
func resolveExecTarget(commandStr, scriptFile string) (*execTarget, error) {
	if commandStr == "" && scriptFile == "" {
		return nil, errors.New("either --cmd/-c or --file/-f must be specified")
	}
	if commandStr != "" && scriptFile != "" {
		return nil, errors.New("--cmd and --file are mutually exclusive")
	}

	if commandStr != "" {
		return &execTarget{
			displayName: commandStr,
			content:     commandStr,
			module:      "shell",
			moduleArgs:  commandStr,
			auditDetail: fmt.Sprintf("cmd=%q", commandStr),
			isScript:    false,
		}, nil
	}

	absPath, err := filepath.Abs(scriptFile)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve script file path: %w", err)
	}
	fi, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("script file not found: %w", err)
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("script path is a directory: %s", scriptFile)
	}

	contentBytes, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read script file: %w", err)
	}
	contentStr := string(contentBytes)
	if strings.TrimSpace(contentStr) == "" {
		return nil, errors.New("script file is empty")
	}

	return &execTarget{
		displayName: scriptFile,
		content:     contentStr,
		module:      "script",
		moduleArgs:  absPath,
		auditDetail: fmt.Sprintf("file=%q", scriptFile),
		isScript:    true,
	}, nil
}

func (c *commandSet) newExecCommand() *cobra.Command {
	var (
		group            string
		commandStr       string
		scriptFile       string
		limit            string
		timeoutSec       int
		dryRun           bool
		yes              bool
		allowDestructive bool
	)
	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Execute an ad-hoc shell command or local script on target hosts",
		Long: `Execute an ad-hoc shell command or local script across the target host group via ansible.

Safety model:
  - safe (read-only) commands run immediately.
  - mutating commands (service restart, file write, ...) require interactive
    confirmation or --yes.
  - dangerous commands (rm -rf, mkfs, shutdown, firewall flush, user/account
    changes, crontab edits, package removal, piped remote scripts, ...) are
    refused unless --allow-destructive is passed AND the user confirms.
  - every invocation is written to the opsvault audit log.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveExecTarget(commandStr, scriptFile)
			if err != nil {
				return err
			}

			assessment := safety.AssessShell(target.content)

			// Always audit the attempt before any prompt or short-circuit.
			detail := fmt.Sprintf("group=%s limit=%q timeout=%ds dry_run=%v level=%s reasons=%q %s",
				group, limit, timeoutSec, dryRun, assessment.Level, assessment.FormatReasons(), target.auditDetail)

			printExecHeader(os.Stdout, group, limit, target.displayName, target.isScript, assessment)

			if dryRun {
				logger.AuditLog("ansible", "exec-dry-run", detail, true)
				fmt.Fprintln(os.Stdout, "[dry-run] no command was sent to remote hosts")
				return nil
			}

			switch assessment.Level {
			case safety.RiskDangerous:
				if !allowDestructive {
					logger.AuditLog("ansible", "exec-blocked", detail, false)
					return fmt.Errorf(
						"refused: command matches destructive pattern %s\n"+
							"If you really intend this, re-run with --allow-destructive and a narrowed --limit.",
						assessment.FormatReasons())
				}
				if !yes && !confirmInteractively(cmd, "DANGEROUS") {
					logger.AuditLog("ansible", "exec-declined", detail, false)
					return errors.New("aborted by user")
				}
			case safety.RiskMutating:
				if !yes && !confirmInteractively(cmd, "MUTATING") {
					logger.AuditLog("ansible", "exec-declined", detail, false)
					return errors.New("aborted by user")
				}
			case safety.RiskSafe:
				// no confirmation needed
			}

			exec, cleanup, err := c.getExecutor()
			if err != nil {
				logger.AuditLog("ansible", "exec-error", detail, false)
				return err
			}
			defer cleanup()

			ctx := cmd.Context()
			if timeoutSec > 0 && target.isScript {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
				defer cancel()
			}

			opts := ansible.AdhocOpts{Limit: limit, TimeoutSec: timeoutSec}
			err = exec.RunAnsibleWithOpts(ctx, group, target.module, target.moduleArgs, opts, os.Stdout, os.Stderr)
			if err != nil {
				logger.AuditLog("ansible", "exec", detail, false)
				return fmt.Errorf("execution failed: %w", err)
			}
			logger.AuditLog("ansible", "exec", detail, true)
			return nil
		},
	}
	cmd.Flags().StringVarP(&group, "group", "g", "all", "target host group")
	cmd.Flags().StringVarP(&commandStr, "cmd", "c", "", "shell command to execute")
	cmd.Flags().StringVarP(&scriptFile, "file", "f", "", "path to local shell script to execute")
	cmd.Flags().StringVar(&limit, "limit", "", "restrict to specific host(s) (ansible -l); strongly recommended for first runs")
	cmd.Flags().IntVar(&timeoutSec, "timeout", 0, "remote command timeout in seconds (0 = no timeout)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the parsed command + risk level without sending to remote hosts")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip interactive confirmation (still blocked for dangerous without --allow-destructive)")
	cmd.Flags().BoolVar(&allowDestructive, "allow-destructive", false, "permit dangerous commands (requires interactive confirmation or --yes)")
	return cmd
}

// printExecHeader renders a short banner before any prompt so the user sees
// exactly what is about to be sent where.
func printExecHeader(out *os.File, group, limit, target string, isScript bool, a safety.Assessment) {
	fmt.Fprintf(out, "\n=== opsvault ansible exec ===\n")
	fmt.Fprintf(out, "  group:   %s\n", group)
	if limit != "" {
		fmt.Fprintf(out, "  limit:   %s\n", limit)
	}
	if isScript {
		fmt.Fprintf(out, "  script:  %s\n", target)
	} else {
		fmt.Fprintf(out, "  command: %s\n", target)
	}
	fmt.Fprintf(out, "  risk:    %s %s\n\n", a.Level, a.FormatReasons())
}

// confirmInteractively prompts the user to type the expected word to proceed.
// Returns true only on exact match. Decline on EOF / mismatch.
func confirmInteractively(cmd *cobra.Command, level string) bool {
	in := cmd.InOrStdin()
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Type %s to confirm execution on the target group: ", level)
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	return strings.TrimSpace(line) == level
}
