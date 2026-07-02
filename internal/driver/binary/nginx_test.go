package binary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func testNginxConfig(t *testing.T) *viper.Viper {
	t.Helper()
	dir := t.TempDir()
	cfg := viper.New()
	cfg.Set("oneinstack.nginx_install_path", filepath.Join(dir, "nginx"))
	cfg.Set("oneinstack.www_root", filepath.Join(dir, "wwwroot"))
	cfg.Set("oneinstack.ssl_root", filepath.Join(dir, "ssl"))
	return cfg
}

func TestAddVHostCreatesRootAndConfig(t *testing.T) {
	cfg := testNginxConfig(t)
	drv := NewNginxDriver(cfg)
	root := filepath.Join(cfg.GetString("oneinstack.www_root"), "example")
	reloads := 0
	oldReload := reloadNginx
	reloadNginx = func() error {
		reloads++
		return nil
	}
	defer func() {
		reloadNginx = oldReload
	}()

	if err := drv.AddVHost("example.com", root); err != nil {
		t.Fatalf("AddVHost: %v", err)
	}

	confPath := filepath.Join(cfg.GetString("oneinstack.nginx_install_path"), "conf", "vhost", "example.com.conf")
	data, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("read conf: %v", err)
	}
	if !strings.Contains(string(data), "server_name example.com;") {
		t.Fatalf("conf missing server_name: %s", string(data))
	}
	if !strings.Contains(string(data), "root "+root+";") {
		t.Fatalf("conf missing root: %s", string(data))
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("stat root: %v", err)
	}
	if reloads != 1 {
		t.Fatalf("reloads after add = %d, want 1", reloads)
	}
}

func TestReloadUsesReloadHook(t *testing.T) {
	cfg := testNginxConfig(t)
	drv := NewNginxDriver(cfg)
	reloads := 0
	oldReload := reloadNginx
	reloadNginx = func() error {
		reloads++
		return nil
	}
	defer func() {
		reloadNginx = oldReload
	}()

	if err := drv.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if reloads != 1 {
		t.Fatalf("reloads = %d, want 1", reloads)
	}
}

func TestEnableAndDisableSSLRewritesVHostConfig(t *testing.T) {
	cfg := testNginxConfig(t)
	drv := NewNginxDriver(cfg)
	root := filepath.Join(cfg.GetString("oneinstack.www_root"), "example")
	reloads := 0
	oldReload := reloadNginx
	reloadNginx = func() error {
		reloads++
		return nil
	}
	defer func() {
		reloadNginx = oldReload
	}()

	if err := drv.AddVHost("example.com", root); err != nil {
		t.Fatalf("AddVHost: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cfg.GetString("oneinstack.ssl_root"), "example.com"), 0o755); err != nil {
		t.Fatalf("mkdir ssl dir: %v", err)
	}

	if err := drv.EnableSSL("example.com"); err != nil {
		t.Fatalf("EnableSSL: %v", err)
	}

	confPath := filepath.Join(cfg.GetString("oneinstack.nginx_install_path"), "conf", "vhost", "example.com.conf")
	sslConf, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("read ssl conf: %v", err)
	}
	sslText := string(sslConf)
	if !strings.Contains(sslText, "listen 443 ssl;") {
		t.Fatalf("ssl conf missing 443 server: %s", sslText)
	}
	if !strings.Contains(sslText, "return 301 https://$host$request_uri;") {
		t.Fatalf("ssl conf missing redirect: %s", sslText)
	}
	if !strings.Contains(sslText, "ssl_certificate "+filepath.Join(cfg.GetString("oneinstack.ssl_root"), "example.com", "fullchain.pem")+";") {
		t.Fatalf("ssl conf missing cert path: %s", sslText)
	}
	if reloads != 2 {
		t.Fatalf("reloads after add+enable = %d, want 2", reloads)
	}

	if err := drv.DisableSSL("example.com"); err != nil {
		t.Fatalf("DisableSSL: %v", err)
	}
	httpConf, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("read http conf: %v", err)
	}
	httpText := string(httpConf)
	if strings.Contains(httpText, "listen 443 ssl;") {
		t.Fatalf("http conf still contains ssl listener: %s", httpText)
	}
	if strings.Contains(httpText, "return 301 https://$host$request_uri;") {
		t.Fatalf("http conf still contains redirect: %s", httpText)
	}
	if !strings.Contains(httpText, "listen 80;") {
		t.Fatalf("http conf missing 80 listener: %s", httpText)
	}
	if reloads != 3 {
		t.Fatalf("reloads after disable = %d, want 3", reloads)
	}
}
