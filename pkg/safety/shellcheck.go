// Package safety provides shell command risk assessment for remote execution.
//
// The goal is not to be a perfect firewall — shell obfuscation can always
// defeat a regex. The goal is to catch *obvious* danger, force explicit
// user confirmation for mutating operations, and refuse known-destructive
// patterns unless the caller passes an explicit override.
package safety

import (
	"fmt"
	"regexp"
	"strings"
)

// RiskLevel classifies how dangerous a shell command is.
type RiskLevel int

const (
	// RiskSafe — read-only inspection (uptime, df, systemctl status, tail, ...).
	RiskSafe RiskLevel = iota
	// RiskMutating — writes to disk, restarts services, manages packages.
	// Requires explicit user confirmation.
	RiskMutating
	// RiskDangerous — known-destructive pattern (rm -rf, mkfs, shutdown, ...).
	// Requires --allow-destructive plus confirmation.
	RiskDangerous
)

func (r RiskLevel) String() string {
	switch r {
	case RiskSafe:
		return "safe"
	case RiskMutating:
		return "mutating"
	case RiskDangerous:
		return "dangerous"
	default:
		return "unknown"
	}
}

// Assessment is the result of inspecting one shell command.
type Assessment struct {
	Level   RiskLevel
	Reasons []string // human-readable explanations for the chosen level
}

// dangerousPatterns are matched case-insensitively against the raw command.
// Each entry carries a short label shown to the user on block.
//
// Deliberately conservative: better to over-flag and require --allow-destructive
// than to under-flag and let `rm -rf /data` through.
var dangerousPatterns = []struct {
	re    *regexp.Regexp
	label string
}{
	// File destruction
	{regexp.MustCompile(`\brm\s+(-[a-zA-Z]*[rR][a-zA-Z]*\s+)?-[a-zA-Z]*f`), "rm with recursive/force flags"},
	{regexp.MustCompile(`\brm\s+.*\s/(\s|$|\*)`), "rm targeting filesystem root"},
	{regexp.MustCompile(`\bshred\b|\bwipefs\b`), "secure-erase tool"},
	{regexp.MustCompile(`\bmkfs(\.\w+)?\b`), "filesystem format"},
	{regexp.MustCompile(`\bdd\b[^|&;]*\bof=/dev/`), "dd writing to a block device"},
	{regexp.MustCompile(`\bfind\b[^|&;]*\s-delete\b`), "find -delete"},
	{regexp.MustCompile(`>\s*/dev/sd[a-z]`), "redirect overwriting a disk device"},

	// Power / disruption
	{regexp.MustCompile(`\b(shutdown|reboot|halt|poweroff)\b`), "power control"},
	{regexp.MustCompile(`\binit\s+[06]\b`), "runlevel change"},
	{regexp.MustCompile(`\bkill\s+-9\s+-1\b|\bkillall\s+-9\b`), "kill all processes"},
	{regexp.MustCompile(`\bpkill\s+-9\s+-u\b`), "kill all user processes"},
	{regexp.MustCompile(`:\(\)\s*\{\s*:\|:&\s*\}\s*;:`), "fork bomb"},

	// Network / firewall
	{regexp.MustCompile(`\biptables\s+-F\b|\biptables\s+--flush\b`), "flush firewall rules"},
	{regexp.MustCompile(`\bnft\s+flush\s+ruleset\b`), "flush nftables ruleset"},

	// Identity / privilege
	{regexp.MustCompile(`\b(useradd|userdel|usermod|passwd|chpasswd)\b`), "user account modification"},
	{regexp.MustCompile(`\bvisudo\b|>\s*/etc/sudoers`), "sudoers modification"},
	{regexp.MustCompile(`\bchmod\s+(-R\s+)?777\b`), "world-writable permission"},
	{regexp.MustCompile(`\bchown\s+-R\b[^|&;]*\s/(\s|$)`), "recursive chown on root"},

	// Persistence / backdoors
	{regexp.MustCompile(`\bcrontab\s+-[er]`), "crontab edit"},
	{regexp.MustCompile(`>>\s*/etc/rc\.local|>>\s*~/\.ssh/authorized_keys|>>\s*\$HOME/\.ssh/authorized_keys`), "persistence hook append"},
	{regexp.MustCompile(`>\s*/etc/systemd/system/`), "systemd unit write"},

	// Mass package ops (should go through opsvault ansible deploy/uninstall)
	{regexp.MustCompile(`\b(yum|dnf|apt|apt-get)\s+(remove|autoremove|erase)\b`), "package removal"},
	{regexp.MustCompile(`\b(yum|dnf|apt|apt-get)\s+(update|upgrade|dist-upgrade)\s+-y\b`), "unattended mass upgrade"},

	// Obfuscation / remote-code-execution
	{regexp.MustCompile(`\bbase64\s+(-d|--decode|-D)\b`), "base64 decode (possible obfuscation)"},
	{regexp.MustCompile(`\beval\s+"?\$\(\s*(curl|wget|fetch)`), "eval of remote-fetched payload"},
	{regexp.MustCompile(`\b(curl|wget)\b[^|&;]*\|\s*(bash|sh|zsh)\b`), "pipe remote script to shell"},
	{regexp.MustCompile(`\$\{IFS\}`), "IFS obfuscation"},
}

// mutatingPatterns mark the command as needing confirmation but not refusal.
var mutatingPatterns = []struct {
	re    *regexp.Regexp
	label string
}{
	{regexp.MustCompile(`\bsystemctl\s+(start|stop|restart|reload|enable|disable|mask|unmask)\b`), "service lifecycle"},
	{regexp.MustCompile(`\bservice\s+\S+\s+(start|stop|restart|reload)\b`), "service lifecycle"},
	{regexp.MustCompile(`\bkill\s+-[A-Z]+\s+\d`), "signal a process"},
	{regexp.MustCompile(`\b(rm|rmdir|mv|cp|touch|mkdir|chmod|chown|chgrp|ln)\b`), "filesystem write"},
	{regexp.MustCompile(`\b(yum|dnf|apt|apt-get)\s+install\b`), "package install"},
	{regexp.MustCompile(`\b(docker|podman|kubectl|systemctl)\s+(rm|stop|restart|kill|delete)\b`), "runtime lifecycle"},
	{regexp.MustCompile(`>\s*[^\s|&;]+`), "shell redirect write"},
	{regexp.MustCompile(`\b(tee|sed\s+-i|truncate|dd)\b`), "file mutation"},
	{regexp.MustCompile(`\b(logrotate|ldconfig|updatedb)\b`), "system state refresh"},
	{regexp.MustCompile(`\b(mount|umount|swapon|swapoff)\b`), "mount state change"},
	{regexp.MustCompile(`\biptables\b|\bnft\b|\bfirewall-cmd\b`), "firewall modification"},
	{regexp.MustCompile(`\bhostnamectl\s+set\b|\btimedatectl\s+set\b`), "host identity change"},
}

// AssessShell inspects one shell command and returns its risk level.
//
// Precedence: any Dangerous match wins over Mutating; any Mutating match wins
// over Safe. Reasons are aggregated across all matching patterns at the
// highest level so the user sees the full picture.
func AssessShell(command string) Assessment {
	command = strings.TrimSpace(command)
	if command == "" {
		return Assessment{Level: RiskSafe, Reasons: []string{"empty command"}}
	}

	var dangerReasons []string
	for _, p := range dangerousPatterns {
		if p.re.MatchString(command) {
			dangerReasons = append(dangerReasons, p.label)
		}
	}
	if len(dangerReasons) > 0 {
		return Assessment{Level: RiskDangerous, Reasons: dangerReasons}
	}

	var mutatingReasons []string
	for _, p := range mutatingPatterns {
		if p.re.MatchString(command) {
			mutatingReasons = append(mutatingReasons, p.label)
		}
	}
	if len(mutatingReasons) > 0 {
		return Assessment{Level: RiskMutating, Reasons: mutatingReasons}
	}

	return Assessment{Level: RiskSafe}
}

// FormatReasons renders the assessment for terminal display.
func (a Assessment) FormatReasons() string {
	if len(a.Reasons) == 0 {
		return ""
	}
	return fmt.Sprintf("[%s]", strings.Join(a.Reasons, ", "))
}
