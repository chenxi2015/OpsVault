package binary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"OpsVault/internal/driver"

	"github.com/spf13/viper"
)

func testNginxConfig(t *testing.T) *viper.Viper {
	t.Helper()
	dir := t.TempDir()
	cfg := viper.New()
	cfg.Set("nginx.install_path", filepath.Join(dir, "nginx"))
	cfg.Set("nginx.www_root", filepath.Join(dir, "wwwroot"))
	cfg.Set("nginx.ssl_root", filepath.Join(dir, "ssl"))
	cfg.Set("nginx.wwwlogs_root", filepath.Join(dir, "wwwlogs"))
	cfg.Set("nginx.run_user", "www")
	cfg.Set("nginx.run_group", "www")
	return cfg
}

func TestNginxInstallPlanMatchesVendoredScriptFlow(t *testing.T) {
	cfg := testNginxConfig(t)
	cfg.Set("nginx.source_root", filepath.Join(t.TempDir(), "src"))
	cfg.Set("nginx.version", "1.31.0")
	cfg.Set("nginx.pcre_version", "8.45")
	cfg.Set("nginx.openssl_version", "1.1.1w")

	plan := newNginxInstallPlan(cfg)

	if plan.installPath != cfg.GetString("nginx.install_path") {
		t.Fatalf("installPath = %q, want config value", plan.installPath)
	}
	if plan.wwwRoot != cfg.GetString("nginx.www_root") {
		t.Fatalf("wwwRoot = %q, want config value", plan.wwwRoot)
	}
	if plan.sslRoot != cfg.GetString("nginx.ssl_root") {
		t.Fatalf("sslRoot = %q, want config value", plan.sslRoot)
	}
	if plan.wwwLogsRoot != cfg.GetString("nginx.wwwlogs_root") {
		t.Fatalf("wwwLogsRoot = %q, want config value", plan.wwwLogsRoot)
	}
	if plan.nginxArchive() != "nginx-1.31.0.tar.gz" {
		t.Fatalf("nginx archive = %q", plan.nginxArchive())
	}
	configure := strings.Join(plan.configureArgs(), " ")
	for _, want := range []string{
		"--prefix=" + cfg.GetString("nginx.install_path"),
		"--user=www",
		"--group=www",
		"--with-http_ssl_module",
		"--with-stream",
		"--with-pcre=../pcre-8.45",
		"--with-openssl=../openssl-1.1.1w",
	} {
		if !strings.Contains(configure, want) {
			t.Fatalf("configure args missing %q: %s", want, configure)
		}
	}
}

func TestAddVHostCreatesRootAndConfig(t *testing.T) {
	cfg := testNginxConfig(t)
	drv := NewNginxDriver(cfg)
	root := filepath.Join(cfg.GetString("nginx.www_root"), "example")
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

	confPath := filepath.Join(cfg.GetString("nginx.install_path"), "conf", "vhost", "example.com.conf")
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

func TestAddVHostDefaultsRoot(t *testing.T) {
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

	if err := drv.AddVHost("defaulted.com", ""); err != nil {
		t.Fatalf("AddVHost: %v", err)
	}

	expectedRoot := filepath.Join(cfg.GetString("nginx.www_root"), "defaulted.com")
	confPath := filepath.Join(cfg.GetString("nginx.install_path"), "conf", "vhost", "defaulted.com.conf")
	data, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("read conf: %v", err)
	}
	if !strings.Contains(string(data), "root "+expectedRoot+";") {
		t.Fatalf("conf missing root: %s", string(data))
	}
	if _, err := os.Stat(expectedRoot); err != nil {
		t.Fatalf("stat root: %v", err)
	}
	if reloads != 1 {
		t.Fatalf("reloads after add = %d, want 1", reloads)
	}
}

func TestDeleteVHostWithCustomRoot(t *testing.T) {
	cfg := testNginxConfig(t)
	drv := NewNginxDriver(cfg)
	customRoot := filepath.Join(t.TempDir(), "custom-root")
	reloads := 0
	oldReload := reloadNginx
	reloadNginx = func() error {
		reloads++
		return nil
	}
	defer func() {
		reloadNginx = oldReload
	}()

	if err := drv.AddVHost("custom.com", customRoot); err != nil {
		t.Fatalf("AddVHost: %v", err)
	}

	if _, err := os.Stat(customRoot); err != nil {
		t.Fatalf("customRoot not created: %v", err)
	}

	if err := drv.DeleteVHost("custom.com", true); err != nil {
		t.Fatalf("DeleteVHost: %v", err)
	}

	if _, err := os.Stat(customRoot); err == nil {
		t.Fatalf("customRoot was not deleted")
	}
	if reloads != 2 {
		t.Fatalf("reloads after add+del = %d, want 2", reloads)
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
	root := filepath.Join(cfg.GetString("nginx.www_root"), "example")
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
	if err := os.MkdirAll(filepath.Join(cfg.GetString("nginx.ssl_root"), "example.com"), 0o755); err != nil {
		t.Fatalf("mkdir ssl dir: %v", err)
	}

	if err := drv.EnableSSL("example.com"); err != nil {
		t.Fatalf("EnableSSL: %v", err)
	}

	confPath := filepath.Join(cfg.GetString("nginx.install_path"), "conf", "vhost", "example.com.conf")
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
	if !strings.Contains(sslText, "ssl_certificate "+filepath.Join(cfg.GetString("nginx.ssl_root"), "example.com", "fullchain.pem")+";") {
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

func TestNginxDriverCapabilities(t *testing.T) {
	cfg := testNginxConfig(t)
	drv := NewNginxDriver(cfg)

	if _, ok := interface{}(drv).(driver.LogReader); !ok {
		t.Errorf("NginxDriver does not implement driver.LogReader")
	}
	if _, ok := interface{}(drv).(driver.Reloadable); !ok {
		t.Errorf("NginxDriver does not implement driver.Reloadable")
	}
	if _, ok := interface{}(drv).(driver.VHostManager); !ok {
		t.Errorf("NginxDriver does not implement driver.VHostManager")
	}
	if _, ok := interface{}(drv).(driver.SSLManager); !ok {
		t.Errorf("NginxDriver does not implement driver.SSLManager")
	}
}

func TestAddVHostProxyCreatesProxyConfig(t *testing.T) {
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

	if err := drv.AddVHostProxy("api.example.com", "8080"); err != nil {
		t.Fatalf("AddVHostProxy: %v", err)
	}

	confPath := filepath.Join(cfg.GetString("nginx.install_path"), "conf", "vhost", "api.example.com.conf")
	data, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("read conf: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "server_name api.example.com;") {
		t.Fatalf("conf missing server_name: %s", text)
	}
	if !strings.Contains(text, "proxy_pass http://127.0.0.1:8080;") {
		t.Fatalf("conf missing proxy_pass: %s", text)
	}
	if !strings.Contains(text, "proxy_set_header Host $host;") {
		t.Fatalf("conf missing proxy headers: %s", text)
	}
	if reloads != 1 {
		t.Fatalf("reloads after add proxy = %d, want 1", reloads)
	}

	// Test EnableSSL for Proxy VHost preserves proxy_pass
	if err := drv.EnableSSL("api.example.com"); err != nil {
		t.Fatalf("EnableSSL for proxy: %v", err)
	}
	sslData, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("read ssl conf: %v", err)
	}
	sslText := string(sslData)
	if !strings.Contains(sslText, "listen 443 ssl;") {
		t.Fatalf("ssl conf missing 443 server: %s", sslText)
	}
	if !strings.Contains(sslText, "proxy_pass http://127.0.0.1:8080;") {
		t.Fatalf("ssl conf lost proxy_pass: %s", sslText)
	}
}
