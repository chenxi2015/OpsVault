package ansiblecmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"OpsVault/internal/driver/ansible"
	"OpsVault/pkg/logger"
	"OpsVault/pkg/safety"

	"github.com/spf13/cobra"
)

func (c *commandSet) newExecCommand() *cobra.Command {
	var (
		group            string
		commandStr       string
		limit            string
		timeoutSec       int
		dryRun           bool
		yes              bool
		allowDestructive bool
	)
	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Execute an ad-hoc shell command on target hosts",
		Long: `Execute an ad-hoc shell command across the target host group via ansible.

Safety model:
  - safe (read-only) commands run immediately.
  - mutating commands (service restart, file write, ...) require interactive
    confirmation or --yes.
  - dangerous commands (rm -rf, mkfs, shutdown, firewall flush, user/account
    changes, crontab edits, package removal, piped remote scripts, ...) are
    refused unless --allow-destructive is passed AND the user confirms.
  - every invocation is written to the opsvault audit log.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if commandStr == "" {
				return errors.New("command string must not be empty, use --cmd")
			}

			assessment := safety.AssessShell(commandStr)

			// Always audit the attempt before any prompt or short-circuit.
			detail := fmt.Sprintf("group=%s limit=%q timeout=%ds dry_run=%v level=%s reasons=%q cmd=%q",
				group, limit, timeoutSec, dryRun, assessment.Level, assessment.FormatReasons(), commandStr)

			printExecHeader(os.Stdout, group, limit, commandStr, assessment)

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

			opts := ansible.AdhocOpts{Limit: limit, TimeoutSec: timeoutSec}
			err = exec.RunAnsibleWithOpts(cmd.Context(), group, "shell", commandStr, opts, os.Stdout, os.Stderr)
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
	cmd.Flags().StringVar(&limit, "limit", "", "restrict to specific host(s) (ansible -l); strongly recommended for first runs")
	cmd.Flags().IntVar(&timeoutSec, "timeout", 0, "remote command timeout in seconds (0 = no timeout)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the parsed command + risk level without sending to remote hosts")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip interactive confirmation (still blocked for dangerous without --allow-destructive)")
	cmd.Flags().BoolVar(&allowDestructive, "allow-destructive", false, "permit dangerous commands (requires interactive confirmation or --yes)")
	_ = cmd.MarkFlagRequired("cmd")
	return cmd
}

// printExecHeader renders a short banner before any prompt so the user sees
// exactly what is about to be sent where.
func printExecHeader(out *os.File, group, limit, cmdStr string, a safety.Assessment) {
	fmt.Fprintf(out, "\n=== opsvault ansible exec ===\n")
	fmt.Fprintf(out, "  group:   %s\n", group)
	if limit != "" {
		fmt.Fprintf(out, "  limit:   %s\n", limit)
	}
	fmt.Fprintf(out, "  command: %s\n", cmdStr)
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
