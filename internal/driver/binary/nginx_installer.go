package binary

import (
	"bytes"
	"compress/gzip"
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
	"OpsVault/pkg/logger"
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
	noStart         bool
	force           bool
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
	rootDir := ""
	if cfg != nil {
		rootDir = cfg.GetString("system.root_dir")
	}
	if rootDir == "" {
		rootDir = "/data/opsvault"
	}

	opensslVer := versionutil.ResolveOpenSSLVersion(configString(cfg, "nginx.openssl_version", "latest"), "3.0.15")

	nginxVer := versionutil.ResolveNginxVersion(
		configString(cfg, "nginx.version", "latest"),
		"1.26.2",
	)

	wwwRoot := configString(cfg, "nginx.www_root", "")
	if wwwRoot == "" {
		wwwRoot = configString(cfg, "nginx.data_root", filepath.Join(rootDir, "wwwroot"))
	}
	sslRoot := configString(cfg, "nginx.ssl_root", filepath.Join(rootDir, "ssl"))
	wwwLogsRoot := configString(cfg, "nginx.wwwlogs_root", "")
	if wwwLogsRoot == "" {
		wwwLogsRoot = configString(cfg, "nginx.datalogs_root", filepath.Join(rootDir, "wwwlogs"))
	}

	return nginxInstallPlan{
		sourceRoot:      configString(cfg, "nginx.source_root", "/usr/local/src/opsvault-nginx"),
		installPath:     configString(cfg, "nginx.install_path", "/usr/local/nginx"),
		wwwRoot:         wwwRoot,
		sslRoot:         sslRoot,
		wwwLogsRoot:     wwwLogsRoot,
		runUser:         configString(cfg, "nginx.run_user", "www"),
		runGroup:        configString(cfg, "nginx.run_group", "www"),
		version:         nginxVer,
		pcreVersion:     configString(cfg, "nginx.pcre_version", "8.45"),
		opensslVersion:  opensslVer,
		modulesOptions:  cfg.GetStringSlice("nginx.modules_options"),
		systemdUnitPath: configString(cfg, "nginx.systemd_unit_path", "/lib/systemd/system/nginx.service"),
		logrotatePath:   configString(cfg, "nginx.logrotate_path", "/etc/logrotate.d/nginx"),
		jobs:            configInt(cfg, "nginx.make_jobs", runtime.NumCPU()),
		noStart:         cfg != nil && cfg.GetBool("nginx.no_start"),
		force:           cfg != nil && cfg.GetBool("nginx.force"),
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

func (p nginxInstallPlan) pcreSourceDir() string {
	return filepath.Join(p.sourceRoot, "pcre-"+p.pcreVersion)
}

func (p nginxInstallPlan) opensslSourceDir() string {
	return filepath.Join(p.sourceRoot, "openssl-"+p.opensslVersion)
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
	logger.Infof("[nginx] Starting binary installation (Nginx v%s, OpenSSL v%s)...", i.plan.version, i.plan.opensslVersion)
	logger.Infof("[nginx] Step 1/8: Installing host dependencies...")
	if err := i.ensureHostDependencies(); err != nil {
		logger.Errorf("[nginx] Failed to install host dependencies: %v", err)
		return err
	}
	logger.Infof("[nginx] Step 2/8: Creating runtime user/group (%s)...", i.plan.runUser)
	if err := i.ensureRuntimeUser(); err != nil {
		logger.Errorf("[nginx] Failed to create runtime user: %v", err)
		return err
	}
	logger.Infof("[nginx] Step 3/8: Preparing directories...")
	if err := i.prepareDirectories(); err != nil {
		logger.Errorf("[nginx] Failed to prepare directories: %v", err)
		return err
	}
	logger.Infof("[nginx] Step 4/8: Downloading source archives...")
	if err := i.downloadSources(); err != nil {
		logger.Errorf("[nginx] Failed to download source archives: %v", err)
		return err
	}
	logger.Infof("[nginx] Step 5/8: Extracting sources...")
	if err := i.extractSources(); err != nil {
		logger.Errorf("[nginx] Failed to extract source archives: %v", err)
		return err
	}
	logger.Infof("[nginx] Step 6/8: Compiling and installing Nginx...")
	if err := i.compileAndInstall(); err != nil {
		logger.Errorf("[nginx] Failed during compile and install: %v", err)
		return err
	}
	logger.Infof("[nginx] Step 7/8: Setting up symlinks and runtime config files...")
	if err := i.ensureSymlinks(); err != nil {
		logger.Errorf("[nginx] Failed to set up symlinks: %v", err)
		return err
	}
	if err := i.writeRuntimeFiles(); err != nil {
		logger.Errorf("[nginx] Failed to write runtime config files: %v", err)
		return err
	}
	logger.Infof("[nginx] Step 8/8: Enabling and starting Nginx systemd service...")
	if err := system.ReloadDaemon(); err != nil {
		logger.Errorf("[nginx] Failed to reload systemd daemon: %v", err)
		return err
	}
	if err := system.EnableService("nginx"); err != nil {
		logger.Errorf("[nginx] Failed to enable Nginx service: %v", err)
		return err
	}
	if i.plan.noStart {
		logger.Infof("[nginx] Skipping Nginx service start (--no-start enabled). Run 'opsvault nginx start' to start manually.")
		logger.Infof("[nginx] Nginx binary installation completed successfully!")
		return nil
	}
	if err := system.StartService("nginx"); err != nil {
		logger.Errorf("[nginx] Failed to start Nginx service: %v", err)
		return err
	}
	logger.Infof("[nginx] Nginx binary installation completed successfully!")
	return nil
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

// isTarGzValid verifies that the given file exists, is non-empty, and is a complete, uncorrupted .tar.gz archive.
// It detects truncated downloads (EOF), invalid headers, and CRC32 checksum errors.
func isTarGzValid(filePath string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return false
	}
	defer gzr.Close()

	// io.Copy to io.Discard reads the entire gzip stream, validating the gzip CRC32 checksum and EOF
	_, err = io.Copy(io.Discard, gzr)
	return err == nil
}

func (i *nginxInstaller) downloadSources() error {
	var proxies []string
	if i.plan.config != nil {
		if p := i.plan.config.GetStringSlice("nginx.github_proxies"); len(p) > 0 {
			proxies = p
		} else if p := i.plan.config.GetStringSlice("system.github_proxies"); len(p) > 0 {
			proxies = p
		}
	}
	if len(proxies) == 0 {
		proxies = versionutil.DefaultGitHubProxies
	}

	opensslCandidateURLs := versionutil.GetOpenSSLDownloadURLs(i.plan.opensslVersion, proxies...)
	if customURL := configString(i.plan.config, "nginx.source_urls.openssl", ""); customURL != "" {
		opensslCandidateURLs = append([]string{customURL}, opensslCandidateURLs...)
	}

	var nginxURLs []string
	if customURL := configString(i.plan.config, "nginx.source_urls.nginx", ""); customURL != "" {
		nginxURLs = append(nginxURLs, customURL)
	}
	nginxMirrors := []string{"https://mirrors.sohu.com/nginx/", "https://nginx.org/download/"}
	if i.plan.config != nil {
		if configured := i.plan.config.GetStringSlice("nginx.mirrors.nginx"); len(configured) > 0 {
			nginxMirrors = configured
		}
	}
	for _, m := range nginxMirrors {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		if !strings.HasSuffix(m, "/") {
			m += "/"
		}
		nginxURLs = append(nginxURLs, m+i.plan.nginxArchive())
	}

	var pcreURLs []string
	if customURL := configString(i.plan.config, "nginx.source_urls.pcre", ""); customURL != "" {
		pcreURLs = append(pcreURLs, customURL)
	}
	pcreMirrors := []string{
		"https://mirrors.aliyun.com/macports/distfiles/pcre/",
		"https://sourceforge.net/projects/pcre/files/pcre/" + i.plan.pcreVersion + "/",
	}
	if i.plan.config != nil {
		if configured := i.plan.config.GetStringSlice("nginx.mirrors.pcre"); len(configured) > 0 {
			pcreMirrors = configured
		}
	}
	for _, m := range pcreMirrors {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		if strings.Contains(m, "sourceforge.net") {
			if !strings.HasSuffix(m, "/") {
				m += "/"
			}
			pcreURLs = append(pcreURLs, m+i.plan.pcreArchive()+"/download")
		} else {
			if !strings.HasSuffix(m, "/") {
				m += "/"
			}
			pcreURLs = append(pcreURLs, m+i.plan.pcreArchive())
		}
	}

	sources := []struct {
		filename string
		urls     []string
	}{
		{
			filename: i.plan.nginxArchive(),
			urls:     nginxURLs,
		},
		{
			filename: i.plan.pcreArchive(),
			urls:     pcreURLs,
		},
		{
			filename: i.plan.opensslArchive(),
			urls:     opensslCandidateURLs,
		},
	}

	for _, item := range sources {
		target := filepath.Join(i.plan.sourceRoot, item.filename)
		if !i.plan.force {
			if info, err := os.Stat(target); err == nil {
				if isTarGzValid(target) {
					logger.Infof("[nginx] Source archive %s already exists and is verified intact (size: %d bytes), skipping download", item.filename, info.Size())
					continue
				}
				logger.Warnf("[nginx] Source archive %s exists but is incomplete or corrupted (size: %d bytes), removing to re-download...", item.filename, info.Size())
				_ = os.Remove(target)
			}
		} else {
			_ = os.Remove(target)
		}
		if err := downloadFile(target, item.urls...); err != nil {
			return fmt.Errorf("failed to download %s: %w", item.filename, err)
		}
		if !isTarGzValid(target) {
			_ = os.Remove(target)
			return fmt.Errorf("downloaded source archive %s is invalid or truncated, please check network connection or download mirrors", item.filename)
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
		archivePath := filepath.Join(i.plan.sourceRoot, archive)
		if !isTarGzValid(archivePath) {
			return fmt.Errorf("source archive %s is incomplete or corrupted (unexpected EOF/bad checksum); please remove '%s' and re-run installation", archive, archivePath)
		}
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
	tmp := target + ".download"
	_ = os.Remove(tmp)
	defer func() {
		_ = os.Remove(tmp)
	}()

	for _, sourceURL := range sourceURLs {
		if sourceURL == "" {
			continue
		}
		_ = os.Remove(tmp)
		log.Printf("[info] downloading %s from %s...", filepath.Base(target), sourceURL)
		if curlPath, err := exec.LookPath("curl"); err == nil {
			cmd := exec.Command(curlPath, "-f", "-L", "--http1.1", "--retry", "3", "--retry-delay", "2", "--connect-timeout", "15", "-sS", "-o", tmp, sourceURL)
			if output, err := cmd.CombinedOutput(); err == nil {
				if isTarGzValid(tmp) {
					if err := os.Rename(tmp, target); err == nil {
						return nil
					}
				}
				lastErr = fmt.Errorf("downloaded file from %s is invalid or incomplete archive", sourceURL)
				_ = os.Remove(tmp)
			} else {
				lastErr = fmt.Errorf("curl download %s: %w (%s)", sourceURL, err, string(output))
				log.Printf("[warn] %v, trying next URL/method...", lastErr)
			}
		} else if wgetPath, err := exec.LookPath("wget"); err == nil {
			cmd := exec.Command(wgetPath, "--tries=3", "--timeout=30", "-q", "-O", tmp, sourceURL)
			if output, err := cmd.CombinedOutput(); err == nil {
				if isTarGzValid(tmp) {
					if err := os.Rename(tmp, target); err == nil {
						return nil
					}
				}
				lastErr = fmt.Errorf("downloaded file from %s is invalid or incomplete archive", sourceURL)
				_ = os.Remove(tmp)
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
		file, err := os.Create(tmp)
		if err != nil {
			resp.Body.Close()
			lastErr = err
			continue
		}
		_, copyErr := io.Copy(file, resp.Body)
		resp.Body.Close()
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil {
			_ = os.Remove(tmp)
			if copyErr != nil {
				lastErr = copyErr
			} else {
				lastErr = closeErr
			}
			continue
		}
		if isTarGzValid(tmp) {
			if err := os.Rename(tmp, target); err == nil {
				return nil
			} else {
				lastErr = err
			}
		} else {
			lastErr = fmt.Errorf("http download %s resulted in invalid or incomplete archive", sourceURL)
		}
		_ = os.Remove(tmp)
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
