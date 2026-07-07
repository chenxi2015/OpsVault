package sslutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Manager struct {
	SSLRoot string
}

// checkCertbot verifies that certbot is installed and returns a friendly error if not.
func checkCertbot() error {
	if _, err := exec.LookPath("certbot"); err != nil {
		return fmt.Errorf("certbot not found. Please install it first:\n" +
			"  CentOS/RHEL: yum install -y certbot  (or: dnf install -y certbot)\n" +
			"  Ubuntu/Debian: apt install -y certbot\n" +
			"  Universal:  snap install --classic certbot")
	}
	return nil
}

func (m Manager) Apply(domain, webroot string) error {
	if domain == "" || webroot == "" {
		return fmt.Errorf("domain and webroot are required")
	}
	if err := checkCertbot(); err != nil {
		return err
	}
	if err := os.MkdirAll(m.SSLRoot, 0o755); err != nil {
		return err
	}
	cmd := exec.Command("certbot", "certonly", "--webroot", "-w", webroot, "-d", domain, "--non-interactive", "--agree-tos", "--register-unsafely-without-email")
	return cmd.Run()
}

func (m Manager) Renew(domain string) error {
	if err := checkCertbot(); err != nil {
		return err
	}
	args := []string{"renew"}
	if domain != "" {
		args = []string{"renew", "--cert-name", domain}
	}
	return exec.Command("certbot", args...).Run()
}

func (m Manager) Delete(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain is required")
	}
	return os.RemoveAll(filepath.Join(m.SSLRoot, domain))
}

