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

func (m Manager) Apply(domain, webroot string) error {
	if domain == "" || webroot == "" {
		return fmt.Errorf("domain and webroot are required")
	}
	if err := os.MkdirAll(m.SSLRoot, 0o755); err != nil {
		return err
	}
	cmd := exec.Command("certbot", "certonly", "--webroot", "-w", webroot, "-d", domain, "--non-interactive", "--agree-tos", "--register-unsafely-without-email")
	return cmd.Run()
}

func (m Manager) Renew(domain string) error {
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
