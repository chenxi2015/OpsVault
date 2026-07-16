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
	"OpsVault/pkg/nginxconf"
	"OpsVault/pkg/versionutil"

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
	opensslVer := versionutil.ResolveOpenSSLVersion(configString(cfg, "nginx.openssl_version", "latest"), "3.0.15")

	nginxVer := versionutil.ResolveNginxVersion(
		configString(cfg, "nginx.version", "latest"),
		"1.26.2",
	)

	return nginxInstallPlan{
		sourceRoot:      configString(cfg, "nginx.source_root", "/usr/local/src/opsvault-nginx"),
		installPath:     configString(cfg, "nginx.install_path", "/usr/local/nginx"),
		wwwRoot:         configString(cfg, "nginx.www_root", "/data/wwwroot"),
		sslRoot:         configString(cfg, "nginx.ssl_root", "/data/ssl"),
		wwwLogsRoot:     configString(cfg, "nginx.wwwlogs_root", "/data/wwwlogs"),
		runUser:         configString(cfg, "nginx.run_user", "www"),
		runGroup:        configString(cfg, "nginx.run_group", "www"),
		version:         nginxVer,
		pcreVersion:     configString(cfg, "nginx.pcre_version", "8.45"),
		opensslVersion:  opensslVer,
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
	// Use getent to silently check group/user existence without polluting logs
	if output, err := runNginxCommand("", "getent", "group", i.plan.runGroup); err != nil {
		if output, err = runNginxCommand("", "groupadd", i.plan.runGroup); err != nil {
			return fmt.Errorf("create nginx group %s: %w: %s", i.plan.runGroup, err, string(output))
		}
	}
	// getent passwd exits non-zero when the user does not exist, no stderr noise
	if output, err := runNginxCommand("", "getent", "passwd", i.plan.runUser); err != nil {
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
		i.plan.opensslArchive(): configString(i.plan.config, "nginx.source_urls.openssl", versionutil.OpenSSLSourceURL(i.plan.opensslVersion)),
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
	cfg := i.plan.toNginxConf()
	files := map[string]string{
		filepath.Join(i.plan.installPath, "conf", "nginx.conf"): nginxconf.RenderBaseConfig(cfg),
		filepath.Join(i.plan.installPath, "conf", "proxy.conf"): nginxconf.RenderProxyConfig(),
		i.plan.systemdUnitPath: nginxconf.RenderSystemdUnit(cfg),
		i.plan.logrotatePath:   nginxconf.RenderLogrotate(cfg),
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

// toNginxConf converts the install plan to the shared nginxconf.Config used
// by pkg/nginxconf renderers.
func (p nginxInstallPlan) toNginxConf() nginxconf.Config {
	return nginxconf.Config{
		InstallPath:     p.installPath,
		WWWRoot:         p.wwwRoot,
		SSLRoot:         p.sslRoot,
		WWWLogsRoot:     p.wwwLogsRoot,
		RunUser:         p.runUser,
		RunGroup:        p.runGroup,
		SystemdUnitPath: p.systemdUnitPath,
		LogrotatePath:   p.logrotatePath,
	}
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
