package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitConfigLoadsExplicitFile(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	content := []byte("docker:\n  network_name: custom-net\n")
	if err := os.WriteFile(cfg, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfgFile = cfg
	t.Cleanup(func() {
		cfgFile = ""
	})

	if err := initConfig(); err != nil {
		t.Fatalf("initConfig: %v", err)
	}

	got := AppConfig().GetString("docker.network_name")
	if got != "custom-net" {
		t.Fatalf("docker.network_name = %q, want %q", got, "custom-net")
	}
}
