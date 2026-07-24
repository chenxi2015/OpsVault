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
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
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
	if err := i.ensureSymlinks(); err != nil {
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

func (i *nginxInstaller) ensureSymlinks() error {
	binPath := filepath.Join(i.plan.installPath, "sbin", "nginx")
	symlinks := []string{"/usr/sbin/nginx", "/usr/local/bin/nginx"}
	for _, target := range symlinks {
		if info, err := os.Lstat(target); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				_ = os.Remove(target)
			} else {
				continue
			}
		}
		_ = fileutil.EnsureDir(filepath.Dir(target), 0o755)
		_ = os.Symlink(binPath, target)
	}
	return nil
}

func getNologinShell() string {
	for _, path := range []string{"/sbin/nologin", "/usr/sbin/nologin", "/bin/false"} {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	if path, err := exec.LookPath("nologin"); err == nil {
		return path
	}
	return "/sbin/nologin"
}

func (i *nginxInstaller) ensureHostDependencies() error {
	if _, err := exec.LookPath("apt-get"); err == nil {
		_, _ = runNginxCommand("", "apt-get", "update", "-y")
		packages := []string{
			"build-essential", "cmake", "autoconf", "tar", "gzip", "patch",
			"zlib1g-dev", "libssl-dev", "libgd-dev", "libperl-dev",
			"net-tools", "wget", "curl", "logrotate", "libjemalloc-dev",
		}
		args := append([]string{"install", "-y"}, packages...)
		if output, err := runNginxCommand("", "apt-get", args...); err != nil {
			return fmt.Errorf("install nginx build dependencies: %w: %s", err, string(output))
		}

		// Try libpcre2-dev (standard on Ubuntu 22.04+/Debian 12+), fallback to libpcre3-dev (older distros)
		if _, err := runNginxCommand("", "apt-get", "install", "-y", "libpcre2-dev"); err != nil {
			_, _ = runNginxCommand("", "apt-get", "install", "-y", "libpcre3-dev")
		}
		return nil
	}

	manager := "yum"
	if _, err := exec.LookPath("dnf"); err == nil {
		manager = "dnf"
	} else if _, err := exec.LookPath("yum"); err != nil {
		return fmt.Errorf("no supported package manager found (apt-get, dnf, or yum required)")
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
	nologin := getNologinShell()
	// getent passwd exits non-zero when the user does not exist, no stderr noise
	if output, err := runNginxCommand("", "getent", "passwd", i.plan.runUser); err != nil {
		if output, err = runNginxCommand("", "useradd", "-g", i.plan.runGroup, "-M", "-s", nologin, i.plan.runUser); err != nil {
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
	opensslCandidateURLs := versionutil.GetOpenSSLDownloadURLs(i.plan.opensslVersion)
	if customURL := configString(i.plan.config, "nginx.source_urls.openssl", ""); customURL != "" {
		opensslCandidateURLs = append([]string{customURL}, opensslCandidateURLs...)
	}

	sources := []struct {
		filename string
		urls     []string
	}{
		{
			filename: i.plan.nginxArchive(),
			urls: []string{
				configString(i.plan.config, "nginx.source_urls.nginx", "https://nginx.org/download/"+i.plan.nginxArchive()),
				"https://mirrors.sohu.com/nginx/" + i.plan.nginxArchive(),
			},
		},
		{
			filename: i.plan.pcreArchive(),
			urls: []string{
				configString(i.plan.config, "nginx.source_urls.pcre", "https://sourceforge.net/projects/pcre/files/pcre/"+i.plan.pcreVersion+"/"+i.plan.pcreArchive()+"/download"),
				"https://mirrors.aliyun.com/macports/distfiles/pcre/" + i.plan.pcreArchive(),
			},
		},
		{
			filename: i.plan.opensslArchive(),
			urls:     opensslCandidateURLs,
		},
	}

	for _, item := range sources {
		target := filepath.Join(i.plan.sourceRoot, item.filename)
		if info, err := os.Stat(target); err == nil && info.Size() > 1000000 {
			continue
		}
		if err := downloadFile(target, item.urls...); err != nil {
			return fmt.Errorf("failed to download %s: %w", item.filename, err)
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

func downloadFile(target string, sourceURLs ...string) error {
	var lastErr error
	for _, sourceURL := range sourceURLs {
		if sourceURL == "" {
			continue
		}
		log.Printf("[info] downloading %s from %s...", filepath.Base(target), sourceURL)
		if curlPath, err := exec.LookPath("curl"); err == nil {
			cmd := exec.Command(curlPath, "-f", "-L", "-C", "-", "--retry", "5", "--retry-delay", "2", "--connect-timeout", "15", "-sS", "-o", target, sourceURL)
			if output, err := cmd.CombinedOutput(); err == nil {
				return nil
			} else {
				lastErr = fmt.Errorf("curl download %s: %w (%s)", sourceURL, err, string(output))
				log.Printf("[warn] %v, trying next URL/method...", lastErr)
			}
		} else if wgetPath, err := exec.LookPath("wget"); err == nil {
			cmd := exec.Command(wgetPath, "-c", "--tries=5", "--timeout=30", "-q", "-O", target, sourceURL)
			if output, err := cmd.CombinedOutput(); err == nil {
				return nil
			} else {
				lastErr = fmt.Errorf("wget download %s: %w (%s)", sourceURL, err, string(output))
				log.Printf("[warn] %v, trying next URL/method...", lastErr)
			}
		}

		// Fallback to Go standard http client
		client := http.Client{Timeout: 30 * time.Minute}
		resp, err := client.Get(sourceURL)
		if err != nil {
			lastErr = fmt.Errorf("http download %s: %w", sourceURL, err)
			continue
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			resp.Body.Close()
			lastErr = fmt.Errorf("http download %s: unexpected status %s", sourceURL, resp.Status)
			continue
		}
		tmp := target + ".tmp"
		file, err := os.Create(tmp)
		if err != nil {
			resp.Body.Close()
			lastErr = err
			continue
		}
		_, copyErr := io.Copy(file, resp.Body)
		resp.Body.Close()
		closeErr := file.Close()
		if copyErr != nil {
			_ = os.Remove(tmp)
			lastErr = copyErr
			continue
		}
		if closeErr != nil {
			_ = os.Remove(tmp)
			lastErr = closeErr
			continue
		}
		if err := os.Rename(tmp, target); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return fmt.Errorf("all download methods and mirrors failed for %s: last error: %v", filepath.Base(target), lastErr)
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
