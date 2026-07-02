package binary

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"OpsVault/internal/driver"
	"OpsVault/internal/oneinstack"
	"OpsVault/internal/system"
	"OpsVault/pkg/fileutil"

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

func (d *NginxDriver) Install() error {
	runner := oneinstack.NewRunner(d.Config)
	if err := runner.InstallNginx(); err != nil {
		return err
	}
	if err := system.EnableService("nginx"); err != nil {
		return err
	}
	return system.ApplyULimit()
}

func (d *NginxDriver) Start() error {
	return system.StartService("nginx")
}

func (d *NginxDriver) Stop() error {
	return system.StopService("nginx")
}

func (d *NginxDriver) Restart() error {
	return system.RestartService("nginx")
}

func (d *NginxDriver) Reload() error {
	return reloadNginx()
}

func (d *NginxDriver) Uninstall(purgeData bool) error {
	runner := oneinstack.NewRunner(d.Config)
	if err := runner.UninstallNginx(); err != nil {
		return err
	}
	if purgeData {
		if err := fileutil.RemoveIfExists(d.Config.GetString("oneinstack.www_root")); err != nil {
			return err
		}
		if err := fileutil.RemoveIfExists(d.Config.GetString("oneinstack.ssl_root")); err != nil {
			return err
		}
	}
	return nil
}

func (d *NginxDriver) Upgrade(targetVersion string) error {
	return oneinstack.NewRunner(d.Config).UpgradeNginx(targetVersion)
}

func (d *NginxDriver) Status() (*driver.ServiceStatus, error) {
	installedPath := d.Config.GetString("oneinstack.nginx_install_path")
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
			"www_root": d.Config.GetString("oneinstack.www_root"),
			"ssl_root": d.Config.GetString("oneinstack.ssl_root"),
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
	confPath := filepath.Join(d.Config.GetString("oneinstack.nginx_install_path"), "conf", "vhost", domain+".conf")
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
	confPath := filepath.Join(d.Config.GetString("oneinstack.nginx_install_path"), "conf", "vhost", domain+".conf")
	if err := os.Remove(confPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if deleteRoot {
		root := filepath.Join(d.Config.GetString("oneinstack.www_root"), domain)
		if err := fileutil.RemoveIfExists(root); err != nil {
			return err
		}
	}
	return reloadNginx()
}

func (d *NginxDriver) ListVHosts() ([]map[string]string, error) {
	vhostDir := filepath.Join(d.Config.GetString("oneinstack.nginx_install_path"), "conf", "vhost")
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
	certDir := filepath.Join(d.Config.GetString("oneinstack.ssl_root"), domain)
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
	return filepath.Join(d.Config.GetString("oneinstack.nginx_install_path"), "conf", "vhost", domain+".conf")
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
