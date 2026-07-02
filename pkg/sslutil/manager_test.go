package sslutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteRemovesDomainDirectory(t *testing.T) {
	root := t.TempDir()
	domainDir := filepath.Join(root, "example.com")
	if err := os.MkdirAll(domainDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := (Manager{SSLRoot: root}).Delete("example.com"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := os.Stat(domainDir); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed, stat err=%v", domainDir, err)
	}
}
