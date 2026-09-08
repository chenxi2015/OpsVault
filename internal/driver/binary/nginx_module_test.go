package binary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPresetModulesMap(t *testing.T) {
	wantPresets := []string{"headers-more", "echo", "fancyindex", "brotli", "cache-purge"}
	for _, name := range wantPresets {
		url, ok := PresetModules[name]
		if !ok || url == "" {
			t.Errorf("PresetModules missing preset %q", name)
		}
	}
}

func TestListAndRemoveModules(t *testing.T) {
	cfg := testNginxConfig(t)
	d := NewNginxDriver(cfg)

	installPath := cfg.GetString("nginx.install_path")
	modulesDir := filepath.Join(installPath, "modules")
	confDir := filepath.Join(installPath, "conf", "modules")

	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatalf("failed to create modules dir: %v", err)
	}
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatalf("failed to create conf dir: %v", err)
	}

	// Create dummy module .so and .conf
	dummySo := filepath.Join(modulesDir, "ngx_http_headers_more_filter_module.so")
	if err := os.WriteFile(dummySo, []byte("fake so data"), 0o644); err != nil {
		t.Fatalf("failed to write dummy so: %v", err)
	}

	dummyConf := filepath.Join(confDir, "headers-more.conf")
	confContent := "load_module modules/ngx_http_headers_more_filter_module.so;\n"
	if err := os.WriteFile(dummyConf, []byte(confContent), 0o644); err != nil {
		t.Fatalf("failed to write dummy conf: %v", err)
	}

	mods, err := d.ListModules()
	if err != nil {
		t.Fatalf("ListModules failed: %v", err)
	}

	if len(mods) != 1 {
		t.Fatalf("expected 1 module, got %d", len(mods))
	}
	if !mods[0].Enabled {
		t.Errorf("expected module to be enabled")
	}
	if mods[0].SoName != "ngx_http_headers_more_filter_module.so" {
		t.Errorf("unexpected SoName: %s", mods[0].SoName)
	}

	// Test RemoveModule
	if err := d.RemoveModule("headers-more"); err != nil {
		t.Fatalf("RemoveModule failed: %v", err)
	}

	if _, err := os.Stat(dummySo); !os.IsNotExist(err) {
		t.Errorf("expected dummy .so to be removed")
	}
	if _, err := os.Stat(dummyConf); !os.IsNotExist(err) {
		t.Errorf("expected dummy .conf to be removed")
	}
}

func TestEnsureModulesImportInConfig(t *testing.T) {
	dir := t.TempDir()
	confPath := filepath.Join(dir, "nginx.conf")

	rawConfig := `user www www;
worker_processes auto;
pid /var/run/nginx.pid;

events {
  worker_connections 1024;
}
`
	if err := os.WriteFile(confPath, []byte(rawConfig), 0o644); err != nil {
		t.Fatalf("failed to write temp nginx.conf: %v", err)
	}

	if err := ensureModulesImportInConfig(confPath); err != nil {
		t.Fatalf("ensureModulesImportInConfig failed: %v", err)
	}

	data, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("failed to read updated nginx.conf: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "include modules/*.conf;") {
		t.Errorf("expected include modules/*.conf; to be inserted into nginx.conf")
	}
}
