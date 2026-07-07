package binary

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"OpsVault/internal/system"
	"OpsVault/pkg/fileutil"

	"github.com/spf13/viper"
)

type nginxInstallPlan struct {
	sourceRoot      string
	installPath     string
	wwwRoot         string
	sslRoot         string
	wwwLogsRoot     string
	runUser         string
	runGroup        string
	version         string
	pcreVersion     string
	opensslVersion  string
	modulesOptions  []string
	systemdUnitPath string
	logrotatePath   string
	jobs            int
	config          *viper.Viper
}

type nginxInstaller struct {
	plan nginxInstallPlan
}

var runNginxCommand = func(dir, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = io.MultiWriter(log.Writer(), &buf)
	cmd.Stderr = io.MultiWriter(log.Writer(), &buf)
	err := cmd.Run()
	return buf.Bytes(), err
}

func newNginxInstaller(cfg *viper.Viper) *nginxInstaller {
	return &nginxInstaller{plan: newNginxInstallPlan(cfg)}
}

func newNginxInstallPlan(cfg *viper.Viper) nginxInstallPlan {
	return nginxInstallPlan{
		sourceRoot:      configString(cfg, "nginx.source_root", "/usr/local/src/opsvault-nginx"),
		installPath:     configString(cfg, "nginx.install_path", "/usr/local/nginx"),
		wwwRoot:         configString(cfg, "nginx.www_root", "/data/wwwroot"),
		sslRoot:         configString(cfg, "nginx.ssl_root", "/data/ssl"),
		wwwLogsRoot:     configString(cfg, "nginx.wwwlogs_root", "/data/wwwlogs"),
		runUser:         configString(cfg, "nginx.run_user", "www"),
		runGroup:        configString(cfg, "nginx.run_group", "www"),
		version:         configString(cfg, "nginx.version", "1.31.0"),
		pcreVersion:     configString(cfg, "nginx.pcre_version", "8.45"),
		opensslVersion:  configString(cfg, "nginx.openssl_version", "1.1.1w"),
		modulesOptions:  cfg.GetStringSlice("nginx.modules_options"),
		systemdUnitPath: configString(cfg, "nginx.systemd_unit_path", "/lib/systemd/system/nginx.service"),
		logrotatePath:   configString(cfg, "nginx.logrotate_path", "/etc/logrotate.d/nginx"),
		jobs:            configInt(cfg, "nginx.make_jobs", runtime.NumCPU()),
		config:          cfg,
	}
}

func (p nginxInstallPlan) nginxArchive() string {
	return "nginx-" + p.version + ".tar.gz"
}

func (p nginxInstallPlan) pcreArchive() string {
	return "pcre-" + p.pcreVersion + ".tar.gz"
}

func (p nginxInstallPlan) opensslArchive() string {
	return "openssl-" + p.opensslVersion + ".tar.gz"
}

func (p nginxInstallPlan) nginxSourceDir() string {
	return filepath.Join(p.sourceRoot, "nginx-"+p.version)
}

func (p nginxInstallPlan) configureArgs() []string {
	args := []string{
		"--prefix=" + p.installPath,
		"--user=" + p.runUser,
		"--group=" + p.runGroup,
		"--with-http_stub_status_module",
		"--with-http_sub_module",
		"--with-http_v2_module",
		"--with-http_ssl_module",
		"--with-stream",
		"--with-stream_ssl_preread_module",
		"--with-stream_ssl_module",
		"--with-http_gzip_static_module",
		"--with-http_realip_module",
		"--with-http_flv_module",
		"--with-http_mp4_module",
		"--with-http_stub_status_module",
		"--with-openssl=../openssl-" + p.opensslVersion,
		"--with-pcre=../pcre-" + p.pcreVersion,
		"--with-pcre-jit",
		"--with-ld-opt=-ljemalloc",
	}
	return append(args, p.modulesOptions...)
}

func (i *nginxInstaller) Install() error {
	if err := i.ensureHostDependencies(); err != nil {
		return err
	}
	if err := i.ensureRuntimeUser(); err != nil {
		return err
	}
	if err := i.prepareDirectories(); err != nil {
		return err
	}
	if err := i.downloadSources(); err != nil {
		return err
	}
	if err := i.extractSources(); err != nil {
		return err
	}
	if err := i.compileAndInstall(); err != nil {
		return err
	}
	if err := i.writeRuntimeFiles(); err != nil {
		return err
	}
	if err := system.ReloadDaemon(); err != nil {
		return err
	}
	if err := system.EnableService("nginx"); err != nil {
		return err
	}
	return system.StartService("nginx")
}

func (i *nginxInstaller) ensureHostDependencies() error {
	manager := "yum"
	if _, err := exec.LookPath("dnf"); err == nil {
		manager = "dnf"
	}
	packages := []string{
		"gcc", "gcc-c++", "make", "cmake", "autoconf", "tar", "gzip", "patch",
		"pcre-devel", "zlib", "zlib-devel", "openssl", "openssl-devel",
		"gd-devel", "perl-devel", "net-tools", "wget", "curl", "logrotate",
		"jemalloc", "jemalloc-devel",
	}
	args := append([]string{"-y", "install"}, packages...)
	if output, err := runNginxCommand("", manager, args...); err != nil {
		return fmt.Errorf("install nginx build dependencies: %w: %s", err, string(output))
	}
	return nil
}

func (i *nginxInstaller) ensureRuntimeUser() error {
	if output, err := runNginxCommand("", "getent", "group", i.plan.runGroup); err != nil {
		if output, err = runNginxCommand("", "groupadd", i.plan.runGroup); err != nil {
			return fmt.Errorf("create nginx group %s: %w: %s", i.plan.runGroup, err, string(output))
		}
	}
	if output, err := runNginxCommand("", "id", "-u", i.plan.runUser); err != nil {
		if output, err = runNginxCommand("", "useradd", "-g", i.plan.runGroup, "-M", "-s", "/sbin/nologin", i.plan.runUser); err != nil {
			return fmt.Errorf("create nginx user %s: %w: %s", i.plan.runUser, err, string(output))
		}
	}
	return nil
}

func (i *nginxInstaller) prepareDirectories() error {
	for _, path := range []string{
		i.plan.sourceRoot,
		i.plan.installPath,
		filepath.Join(i.plan.installPath, "conf", "vhost"),
		filepath.Join(i.plan.wwwRoot, "default"),
		i.plan.sslRoot,
		i.plan.wwwLogsRoot,
	} {
		if err := fileutil.EnsureDir(path, 0o755); err != nil {
			return err
		}
	}
	indexPath := filepath.Join(i.plan.wwwRoot, "default", "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return os.WriteFile(indexPath, []byte("<h1>OpsVault Nginx</h1>\n"), 0o644)
	}
	return nil
}

func (i *nginxInstaller) downloadSources() error {
	sources := map[string]string{
		i.plan.nginxArchive():   configString(i.plan.config, "nginx.source_urls.nginx", "https://nginx.org/download/"+i.plan.nginxArchive()),
		i.plan.pcreArchive():    configString(i.plan.config, "nginx.source_urls.pcre", "https://sourceforge.net/projects/pcre/files/pcre/"+i.plan.pcreVersion+"/"+i.plan.pcreArchive()+"/download"),
		i.plan.opensslArchive(): configString(i.plan.config, "nginx.source_urls.openssl", opensslSourceURL(i.plan.opensslVersion)),
	}
	for filename, sourceURL := range sources {
		target := filepath.Join(i.plan.sourceRoot, filename)
		if info, err := os.Stat(target); err == nil && info.Size() > 0 {
			continue
		}
		if err := downloadFile(target, sourceURL); err != nil {
			return err
		}
	}
	return nil
}

func (i *nginxInstaller) extractSources() error {
	for _, dir := range []string{
		filepath.Join(i.plan.sourceRoot, "nginx-"+i.plan.version),
		filepath.Join(i.plan.sourceRoot, "pcre-"+i.plan.pcreVersion),
		filepath.Join(i.plan.sourceRoot, "openssl-"+i.plan.opensslVersion),
	} {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}
	for _, archive := range []string{i.plan.pcreArchive(), i.plan.nginxArchive(), i.plan.opensslArchive()} {
		output, err := runNginxCommand(i.plan.sourceRoot, "tar", "xzf", archive)
		if err != nil {
			return fmt.Errorf("extract %s: %w: %s", archive, err, string(output))
		}
	}
	return nil
}

func (i *nginxInstaller) compileAndInstall() error {
	nginxBin := filepath.Join(i.plan.installPath, "sbin", "nginx")
	backupBin := nginxBin + ".bak"

	// Back up existing binary before recompiling so we can roll back on failure
	if _, err := os.Stat(nginxBin); err == nil {
		if err := copyFile(nginxBin, backupBin); err != nil {
			return fmt.Errorf("backup existing nginx binary: %w", err)
		}
	}

	gccAuto := filepath.Join(i.plan.nginxSourceDir(), "auto", "cc", "gcc")
	if data, err := os.ReadFile(gccAuto); err == nil {
		updated := strings.ReplaceAll(string(data), `CFLAGS="$CFLAGS -g"`, `#CFLAGS="$CFLAGS -g"`)
		if err := os.WriteFile(gccAuto, []byte(updated), 0o644); err != nil {
			return err
		}
	}
	output, err := runNginxCommand(i.plan.nginxSourceDir(), "./configure", i.plan.configureArgs()...)
	if err != nil {
		_ = restoreBackup(nginxBin, backupBin)
		return fmt.Errorf("configure nginx: %w: %s", err, string(output))
	}
	output, err = runNginxCommand(i.plan.nginxSourceDir(), "make", "-j", fmt.Sprintf("%d", i.plan.jobs))
	if err != nil {
		_ = restoreBackup(nginxBin, backupBin)
		return fmt.Errorf("make nginx: %w: %s", err, string(output))
	}
	output, err = runNginxCommand(i.plan.nginxSourceDir(), "make", "install")
	if err != nil {
		_ = restoreBackup(nginxBin, backupBin)
		return fmt.Errorf("make install nginx: %w: %s", err, string(output))
	}
	if _, err := os.Stat(nginxBin); err != nil {
		_ = restoreBackup(nginxBin, backupBin)
		return fmt.Errorf("nginx binary not found after install: %w", err)
	}
	// New binary is confirmed good — remove the backup
	_ = os.Remove(backupBin)
	return nil
}

// copyFile copies src to dst, preserving file permissions.
func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, info.Mode())
}

// restoreBackup replaces target with the backup file if backup exists.
func restoreBackup(target, backup string) error {
	if _, err := os.Stat(backup); os.IsNotExist(err) {
		return nil
	}
	return os.Rename(backup, target)
}

func (i *nginxInstaller) writeRuntimeFiles() error {
	files := map[string]string{
		filepath.Join(i.plan.installPath, "conf", "nginx.conf"): renderNginxBaseConfig(i.plan),
		filepath.Join(i.plan.installPath, "conf", "proxy.conf"): renderNginxProxyConfig(),
		i.plan.systemdUnitPath: renderNginxSystemdService(i.plan),
		i.plan.logrotatePath:   renderNginxLogrotateConfig(i.plan),
	}
	for path, content := range files {
		if err := fileutil.EnsureDir(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func downloadFile(target, sourceURL string) error {
	client := http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Get(sourceURL)
	if err != nil {
		return fmt.Errorf("download %s: %w", sourceURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("download %s: unexpected status %s", sourceURL, resp.Status)
	}
	tmp := target + ".tmp"
	file, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, resp.Body)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, target)
}

func opensslSourceURL(version string) string {
	if strings.HasPrefix(version, "1.1.") {
		return "https://www.openssl.org/source/old/1.1.1/openssl-" + version + ".tar.gz"
	}
	return "https://www.openssl.org/source/openssl-" + version + ".tar.gz"
}

func renderNginxBaseConfig(plan nginxInstallPlan) string {
	return fmt.Sprintf(`user %s %s;
worker_processes auto;

error_log %s/error_nginx.log crit;
pid /var/run/nginx.pid;
worker_rlimit_nofile 51200;

events {
  use epoll;
  worker_connections 51200;
  multi_accept on;
}

http {
  include mime.types;
  default_type application/octet-stream;
  server_names_hash_bucket_size 128;
  client_header_buffer_size 32k;
  large_client_header_buffers 4 32k;
  client_max_body_size 1024m;
  client_body_buffer_size 10m;
  sendfile on;
  tcp_nopush on;
  keepalive_timeout 120;
  server_tokens off;
  tcp_nodelay on;

  gzip on;
  gzip_buffers 16 8k;
  gzip_comp_level 6;
  gzip_http_version 1.1;
  gzip_min_length 256;
  gzip_proxied any;
  gzip_vary on;
  gzip_types text/xml application/xml application/atom+xml application/rss+xml application/xhtml+xml image/svg+xml text/javascript application/javascript application/x-javascript text/x-json application/json text/css text/plain image/x-icon;
  gzip_disable "MSIE [1-6]\.(?!.*SV1)";

  log_format json escape=json '{"@timestamp":"$time_iso8601","server_addr":"$server_addr","remote_addr":"$remote_addr","scheme":"$scheme","request_method":"$request_method","request_uri":"$request_uri","request_time":$request_time,"body_bytes_sent":$body_bytes_sent,"status":"$status","host":"$host","http_referer":"$http_referer","http_user_agent":"$http_user_agent"}';

  server {
    listen 80;
    server_name _;
    access_log %s/access_nginx.log combined;
    root %s/default;
    index index.html index.htm;

    location /nginx_status {
      stub_status on;
      access_log off;
      allow 127.0.0.1;
      deny all;
    }

    location ~ .*\.(gif|jpg|jpeg|png|bmp|swf|flv|mp4|ico)$ {
      expires 30d;
      access_log off;
    }

    location ~ .*\.(js|css)?$ {
      expires 7d;
      access_log off;
    }

    location ~ ^/(\.user.ini|\.ht|\.git|\.svn|\.project|LICENSE|README.md) {
      deny all;
    }

    location /.well-known {
      allow all;
    }
  }

  include vhost/*.conf;
}
`, plan.runUser, plan.runGroup, plan.wwwLogsRoot, plan.wwwLogsRoot, plan.wwwRoot)
}

func renderNginxProxyConfig() string {
	return `proxy_connect_timeout 300s;
proxy_send_timeout 900;
proxy_read_timeout 900;
proxy_buffer_size 32k;
proxy_buffers 4 64k;
proxy_busy_buffers_size 128k;
proxy_redirect off;
proxy_hide_header Vary;
proxy_set_header Accept-Encoding '';
proxy_set_header Referer $http_referer;
proxy_set_header Cookie $http_cookie;
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
`
}

func renderNginxSystemdService(plan nginxInstallPlan) string {
	nginxBin := filepath.Join(plan.installPath, "sbin", "nginx")
	nginxConf := filepath.Join(plan.installPath, "conf", "nginx.conf")
	return fmt.Sprintf(`[Unit]
Description=nginx - high performance web server
Documentation=http://nginx.org/en/docs/
After=network.target

[Service]
Type=forking
PIDFile=/var/run/nginx.pid
ExecStartPost=/bin/sleep 0.1
ExecStartPre=%s -t -c %s
ExecStart=%s -c %s
ExecReload=/bin/kill -s HUP $MAINPID
ExecStop=/bin/kill -s QUIT $MAINPID
TimeoutStartSec=120
LimitNOFILE=1000000
LimitNPROC=1000000
LimitCORE=1000000

[Install]
WantedBy=multi-user.target
`, nginxBin, nginxConf, nginxBin, nginxConf)
}

func renderNginxLogrotateConfig(plan nginxInstallPlan) string {
	return fmt.Sprintf(`%s/*nginx.log {
  daily
  rotate 5
  missingok
  dateext
  compress
  notifempty
  sharedscripts
  postrotate
    [ -e /var/run/nginx.pid ] && kill -USR1 $(cat /var/run/nginx.pid)
  endscript
}
`, plan.wwwLogsRoot)
}

func configString(cfg *viper.Viper, key, fallback string) string {
	if cfg == nil {
		return fallback
	}
	value := cfg.GetString(key)
	if value == "" {
		return fallback
	}
	return value
}

func configInt(cfg *viper.Viper, key string, fallback int) int {
	if cfg == nil {
		return fallback
	}
	value := cfg.GetInt(key)
	if value <= 0 {
		return fallback
	}
	return value
}
