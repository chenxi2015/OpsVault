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
		resetConfigForTest()
	})

	if err := initConfig(); err != nil {
		t.Fatalf("initConfig: %v", err)
	}

	got := AppConfig().GetString("docker.network_name")
	if got != "custom-net" {
		t.Fatalf("docker.network_name = %q, want %q", got, "custom-net")
	}
}

func TestFallbackPathsDeriveFromRootDir(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	content := []byte(`system:
  root_dir: /www/opsvault
nginx:
  www_root: ""
  ssl_root: ""
  wwwlogs_root: ""
`)
	if err := os.WriteFile(cfg, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfgFile = cfg
	t.Cleanup(func() {
		cfgFile = ""
		resetConfigForTest()
	})

	if err := initConfig(); err != nil {
		t.Fatalf("initConfig: %v", err)
	}

	appCfg := AppConfig()
	if got := appCfg.GetString("system.root_dir"); got != "/www/opsvault" {
		t.Errorf("system.root_dir = %q, want %q", got, "/www/opsvault")
	}
	if got := appCfg.GetString("docker.data_root"); got != "/www/opsvault" {
		t.Errorf("docker.data_root = %q, want %q", got, "/www/opsvault")
	}
	expectedWWWRoot := filepath.Join("/www/opsvault", "wwwroot")
	if got := appCfg.GetString("nginx.www_root"); got != expectedWWWRoot {
		t.Errorf("nginx.www_root = %q, want %q", got, expectedWWWRoot)
	}
	expectedSSLRoot := filepath.Join("/www/opsvault", "ssl")
	if got := appCfg.GetString("nginx.ssl_root"); got != expectedSSLRoot {
		t.Errorf("nginx.ssl_root = %q, want %q", got, expectedSSLRoot)
	}
	expectedLogsRoot := filepath.Join("/www/opsvault", "wwwlogs")
	if got := appCfg.GetString("nginx.wwwlogs_root"); got != expectedLogsRoot {
		t.Errorf("nginx.wwwlogs_root = %q, want %q", got, expectedLogsRoot)
	}
}

func TestDefaultYamlPathsDerivation(t *testing.T) {
	cfgFile = filepath.Join("..", "configs", "default.yaml")
	t.Cleanup(func() {
		cfgFile = ""
		resetConfigForTest()
	})

	if err := initConfig(); err != nil {
		t.Fatalf("initConfig on default.yaml: %v", err)
	}

	appCfg := AppConfig()
	rootDir := appCfg.GetString("system.root_dir")
	if rootDir != "/www/opsvault" {
		t.Errorf("expected system.root_dir from default.yaml to be '/www/opsvault', got %q", rootDir)
	}
	if got := appCfg.GetString("docker.data_root"); got != rootDir {
		t.Errorf("expected docker.data_root to inherit system.root_dir %q, got %q", rootDir, got)
	}
	if got := appCfg.GetString("nginx.www_root"); got != filepath.Join(rootDir, "wwwroot") {
		t.Errorf("expected nginx.www_root to be %q, got %q", filepath.Join(rootDir, "wwwroot"), got)
	}
	if got := appCfg.GetString("nginx.wwwlogs_root"); got != filepath.Join(rootDir, "wwwlogs") {
		t.Errorf("expected nginx.wwwlogs_root to be %q, got %q", filepath.Join(rootDir, "wwwlogs"), got)
	}
	if got := appCfg.GetString("nginx.ssl_root"); got != filepath.Join(rootDir, "ssl") {
		t.Errorf("expected nginx.ssl_root to be %q, got %q", filepath.Join(rootDir, "ssl"), got)
	}
}


