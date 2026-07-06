package binary

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"OpsVault/internal/driver"
	"OpsVault/internal/system"
	"OpsVault/pkg/fileutil"
	"OpsVault/pkg/sslutil"
	"OpsVault/pkg/sysutil"

	"github.com/spf13/viper"
)

type NginxDriver struct {
	*BaseDriver
}

var reloadNginx = func() error {
	return system.ReloadService("nginx")
}

func NewNginxDriver(cfg *viper.Viper) *NginxDriver {
	return &NginxDriver{BaseDriver: NewBaseDriver("nginx", cfg)}
}

func (d *NginxDriver) isLinuxOrTest() bool {
	return flag.Lookup("test.v") != nil || sysutil.IsLinux()
}

func (d *NginxDriver) Install() error {
	if !d.isLinuxOrTest() {
		return fmt.Errorf("nginx binary installation is only supported on Linux (CentOS/CentOS Stream)")
	}
	if err := newNginxInstaller(d.Config).Install(); err != nil {
		return err
	}
	return system.ApplyULimit()
}

func (d *NginxDriver) Start() error {
	if !d.isLinuxOrTest() {
		return fmt.Errorf("starting nginx service is only supported on Linux")
	}
	return system.StartService("nginx")
}

func (d *NginxDriver) Stop() error {
	if !d.isLinuxOrTest() {
		return fmt.Errorf("stopping nginx service is only supported on Linux")
	}
	return system.StopService("nginx")
}

func (d *NginxDriver) Restart() error {
	if !d.isLinuxOrTest() {
		return fmt.Errorf("restarting nginx service is only supported on Linux")
	}
	return system.RestartService("nginx")
}

func (d *NginxDriver) Reload() error {
	if !d.isLinuxOrTest() {
		return fmt.Errorf("reloading nginx service is only supported on Linux")
	}
	return reloadNginx()
}

func (d *NginxDriver) Uninstall(purgeData bool) error {
	if !d.isLinuxOrTest() {
		return fmt.Errorf("uninstalling nginx binary is only supported on Linux")
	}
	_ = system.StopService("nginx")
	_ = system.DisableService("nginx")
	plan := newNginxInstallPlan(d.Config)
	for _, path := range []string{plan.installPath, plan.systemdUnitPath, plan.logrotatePath} {
		if err := fileutil.RemoveIfExists(path); err != nil {
			return err
		}
	}
	if purgeData {
		if err := fileutil.RemoveIfExists(nginxConfigString(d.Config, "nginx.www_root")); err != nil {
			return err
		}
		if err := fileutil.RemoveIfExists(nginxConfigString(d.Config, "nginx.ssl_root")); err != nil {
			return err
		}
		if err := fileutil.RemoveIfExists(nginxConfigString(d.Config, "nginx.wwwlogs_root")); err != nil {
			return err
		}
	}
	return system.ReloadDaemon()
}

func (d *NginxDriver) Upgrade(targetVersion string) error {
	if !d.isLinuxOrTest() {
		return fmt.Errorf("upgrading nginx binary is only supported on Linux")
	}
	if targetVersion == "" {
		return fmt.Errorf("target version is required")
	}
	d.Config.Set("nginx.version", targetVersion)
	if err := newNginxInstaller(d.Config).Install(); err != nil {
		return err
	}
	return system.RestartService("nginx")
}

func (d *NginxDriver) Status() (*driver.ServiceStatus, error) {
	installedPath := nginxConfigString(d.Config, "nginx.install_path")
	pid, _ := system.FindPID("nginx")
	status := &driver.ServiceStatus{
		Name:      "nginx",
		Mode:      driver.ModeBinary,
		Status:    "stopped",
		DataPath:  installedPath,
		PID:       pid,
		Ports:     []string{"80", "443"},
		UpdatedAt: time.Now(),
		Details: map[string]string{
			"www_root": nginxConfigString(d.Config, "nginx.www_root"),
			"ssl_root": nginxConfigString(d.Config, "nginx.ssl_root"),
		},
	}
	if pid > 0 {
		status.Running = true
		status.Status = "running"
	}
	if _, err := os.Stat(installedPath); err != nil {
		status.Status = "not installed"
	}
	return status, nil
}

func (d *NginxDriver) AddVHost(domain, root string) error {
	if domain == "" || root == "" {
		return fmt.Errorf("domain and root are required")
	}
	if err := fileutil.EnsureDir(root, 0o755); err != nil {
		return err
	}
	confPath := filepath.Join(nginxConfigString(d.Config, "nginx.install_path"), "conf", "vhost", domain+".conf")
	if err := fileutil.EnsureDir(filepath.Dir(confPath), 0o755); err != nil {
		return err
	}
	conf := renderHTTPVHost(domain, root)
	if err := os.WriteFile(confPath, []byte(conf), 0o644); err != nil {
		return err
	}
	return reloadNginx()
}

func (d *NginxDriver) DeleteVHost(domain string, deleteRoot bool) error {
	if domain == "" {
		return fmt.Errorf("domain is required")
	}
	confPath := filepath.Join(nginxConfigString(d.Config, "nginx.install_path"), "conf", "vhost", domain+".conf")
	if err := os.Remove(confPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if deleteRoot {
		root := filepath.Join(nginxConfigString(d.Config, "nginx.www_root"), domain)
		if err := fileutil.RemoveIfExists(root); err != nil {
			return err
		}
	}
	return reloadNginx()
}

func (d *NginxDriver) ListVHosts() ([]map[string]string, error) {
	vhostDir := filepath.Join(nginxConfigString(d.Config, "nginx.install_path"), "conf", "vhost")
	entries, err := os.ReadDir(vhostDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var result []map[string]string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		result = append(result, map[string]string{
			"domain": entry.Name(),
			"path":   filepath.Join(vhostDir, entry.Name()),
		})
	}
	return result, nil
}

func (d *NginxDriver) EnableSSL(domain string) error {
	confPath := d.vhostConfPath(domain)
	data, err := os.ReadFile(confPath)
	if err != nil {
		return err
	}
	root := extractRootPath(string(data))
	if root == "" {
		return fmt.Errorf("failed to extract root from %s", confPath)
	}
	certDir := filepath.Join(nginxConfigString(d.Config, "nginx.ssl_root"), domain)
	fullchain := filepath.Join(certDir, "fullchain.pem")
	privkey := filepath.Join(certDir, "privkey.pem")
	conf := renderSSLVHost(domain, root, fullchain, privkey)
	if err := os.WriteFile(confPath, []byte(conf), 0o644); err != nil {
		return err
	}
	return reloadNginx()
}

func (d *NginxDriver) DisableSSL(domain string) error {
	confPath := d.vhostConfPath(domain)
	data, err := os.ReadFile(confPath)
	if err != nil {
		return err
	}
	root := extractRootPath(string(data))
	if root == "" {
		return fmt.Errorf("failed to extract root from %s", confPath)
	}
	if err := os.WriteFile(confPath, []byte(renderHTTPVHost(domain, root)), 0o644); err != nil {
		return err
	}
	return reloadNginx()
}

func (d *NginxDriver) vhostConfPath(domain string) string {
	return filepath.Join(nginxConfigString(d.Config, "nginx.install_path"), "conf", "vhost", domain+".conf")
}

func nginxConfigString(cfg *viper.Viper, key string) string {
	switch key {
	case "nginx.install_path":
		return configString(cfg, key, "/usr/local/nginx")
	case "nginx.www_root":
		return configString(cfg, key, "/data/wwwroot")
	case "nginx.ssl_root":
		return configString(cfg, key, "/data/ssl")
	case "nginx.wwwlogs_root":
		return configString(cfg, key, "/data/wwwlogs")
	default:
		return configString(cfg, key, "")
	}
}

func renderHTTPVHost(domain, root string) string {
	return fmt.Sprintf("server {\n    listen 80;\n    server_name %s;\n    root %s;\n    index index.html index.htm;\n}\n", domain, root)
}

func renderSSLVHost(domain, root, fullchain, privkey string) string {
	return fmt.Sprintf(
		"server {\n    listen 80;\n    server_name %s;\n    return 301 https://$host$request_uri;\n}\n\nserver {\n    listen 443 ssl;\n    server_name %s;\n    root %s;\n    index index.html index.htm;\n    ssl_certificate %s;\n    ssl_certificate_key %s;\n}\n",
		domain, domain, root, fullchain, privkey,
	)
}

func extractRootPath(conf string) string {
	re := regexp.MustCompile(`(?m)^\s*root\s+([^;]+);`)
	matches := re.FindStringSubmatch(conf)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func (d *NginxDriver) TailLogs(lines int) (string, error) {
	journal := exec.Command("journalctl", "-u", "nginx", "-n", strconv.Itoa(lines), "--no-pager")
	out, err := journal.CombinedOutput()
	return string(out), err
}

func (d *NginxDriver) ApplySSL(domain string) error {
	if !d.isLinuxOrTest() {
		return fmt.Errorf("applying SSL certificates is only supported on Linux")
	}
	root := filepath.Join(nginxConfigString(d.Config, "nginx.www_root"), domain)
	manager := sslutil.Manager{SSLRoot: nginxConfigString(d.Config, "nginx.ssl_root")}
	if err := manager.Apply(domain, root); err != nil {
		return err
	}
	return d.EnableSSL(domain)
}

func (d *NginxDriver) RenewSSL(domain string) error {
	if !d.isLinuxOrTest() {
		return fmt.Errorf("renewing SSL certificates is only supported on Linux")
	}
	manager := sslutil.Manager{SSLRoot: nginxConfigString(d.Config, "nginx.ssl_root")}
	if err := manager.Renew(domain); err != nil {
		return err
	}
	return d.Reload()
}

func (d *NginxDriver) DeleteSSL(domain string) error {
	if !d.isLinuxOrTest() {
		return fmt.Errorf("deleting SSL certificates is only supported on Linux")
	}
	manager := sslutil.Manager{SSLRoot: nginxConfigString(d.Config, "nginx.ssl_root")}
	if err := manager.Delete(domain); err != nil {
		return err
	}
	return d.DisableSSL(domain)
}
